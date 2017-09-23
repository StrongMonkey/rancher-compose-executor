package handlers

import (
	"errors"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
	"github.com/rancher/rancher-compose-executor/resources/service"
)

func CreateStack(event *events.Event, apiClient *client.RancherClient) error {
	return doUp(event, apiClient, "Create Stack", true)
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
		if project.IsErrClusterNotReady(err) {
			publishTransitioningReply("Waiting for cluster to be ready", event, apiClient, false)
			return nil
		}
		logger.Errorf("%s Event Failed: %v", msg, err)
		if err != service.ErrTimeout {
			publishTransitioningReply(err.Error(), event, apiClient, true)
		}
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

	cluster, err := apiClient.Cluster.ById(stack.ClusterId)
	if err != nil {
		return err
	}
	if cluster == nil {
		return errors.New("Failed to find cluster")
	}

	project, err := constructProject(stack, cluster, *apiClient.GetOpts())
	if err != nil || project == nil {
		return err
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
