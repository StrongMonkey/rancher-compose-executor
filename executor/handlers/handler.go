package handlers

import (
	"errors"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/project/options"
)

func CreateStack(event *events.Event, apiClient *client.RancherClient) error {
	return doUp(event, apiClient, "Create Stack", false)
}

func UpdateStack(event *events.Event, apiClient *client.RancherClient) error {
	return doUp(event, apiClient, "Update Stack", true)
}

func doUp(event *events.Event, apiClient *client.RancherClient, msg string, forceUp bool) error {
	logger := logrus.WithFields(logrus.Fields{
		"resourceId": event.ResourceID,
		"eventId":    event.ID,
	})

	logger.Infof("%s Event Received", msg)

	if err := stackUp(event, apiClient, forceUp); err != nil {
		logger.Errorf("%s Event Failed: %v", msg, err)
		publishTransitioningReply(err.Error(), event, apiClient, true)
		return err
	}

	logger.Infof("%s Event Done", msg)
	return emptyReply(event, apiClient)
}

func stackUp(event *events.Event, apiClient *client.RancherClient, forceUp bool) error {
	stack, err := apiClient.Stack.ById(event.ResourceID)
	if err != nil {
		return err
	}

	if stack == nil {
		return errors.New("Failed to find stack")
	}

	project, err := constructProject(stack, *apiClient.GetOpts())
	if err != nil || project == nil {
		return err
	}

	if stack.Prune {
		existingServices := map[string]*client.Service{}
		for _, serviceId := range stack.ServiceIds {
			service, err := apiClient.Service.ById(serviceId)
			if err != nil {
				return err
			}
			existingServices[service.Name] = service
		}
		for name, svc := range existingServices {
			if _, ok := project.Config.Services[name]; !ok {
				logrus.Infof("Pruning stack now. Deleting service %v", name)
				if err := apiClient.Service.Delete(svc); err != nil {
					return err
				}
			}
		}
	}

	publishTransitioningReply("Creating stack", event, apiClient, false)

	defer keepalive(event, apiClient)()

	if err := project.Create(context.Background(), options.Options{}); err != nil {
		return err
	}

	if forceUp {
		return project.Up(context.Background(), options.Options{})
	}

	return nil
}
