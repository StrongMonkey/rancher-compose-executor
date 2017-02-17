package handlers

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/rancher-compose/lookup"
	"github.com/rancher/rancher-compose/project"
	"github.com/rancher/rancher-compose/rancher"
)

func constructProjectUpgrade(logger *logrus.Entry, stack *client.Stack, upgradeOpts client.StackUpgrade, url, accessKey, secretKey string) (*project.Project, map[string]interface{}, error) {
	variables, err := createVariableMap(stack, upgradeOpts.RancherCompose)
	if err != nil {
		return nil, nil, err
	}

	for k, v := range upgradeOpts.Environment {
		variables[k] = v
	}

	context := rancher.Context{
		Context: project.Context{
			ProjectName: stack.Name,
			ComposeBytes: [][]byte{
				[]byte(upgradeOpts.DockerCompose),
			},
			ResourceLookup: &lookup.FileResourceLookup{},
			EnvironmentLookup: &lookup.MapEnvLookup{
				Env: variables,
			},
		},
		Url:                 fmt.Sprintf("%s/projects/%s/schemas", url, stack.AccountId),
		AccessKey:           accessKey,
		SecretKey:           secretKey,
		RancherComposeBytes: []byte(upgradeOpts.RancherCompose),
		Upgrade:             true,
	}

	p, err := rancher.NewProject(&context)
	if err != nil {
		return nil, nil, err
	}

	p.AddListener(NewListenLogger(logger, p))
	return p, variables, nil
}

func constructProject(logger *logrus.Entry, stack *client.Stack, url, accessKey, secretKey string) (*rancher.Context, *project.Project, error) {
	variables, err := createVariableMap(stack, stack.RancherCompose)
	if err != nil {
		return nil, nil, err
	}

	context := rancher.Context{
		Context: project.Context{
			ProjectName: stack.Name,
			ComposeBytes: [][]byte{
				[]byte(stack.DockerCompose),
			},
			ResourceLookup: &lookup.FileResourceLookup{},
			EnvironmentLookup: &lookup.MapEnvLookup{
				Env: variables,
			},
		},
		Url:                 fmt.Sprintf("%s/projects/%s/schemas", url, stack.AccountId),
		AccessKey:           accessKey,
		SecretKey:           secretKey,
		RancherComposeBytes: []byte(stack.RancherCompose),
	}

	p, err := rancher.NewProject(&context)
	if err != nil {
		return nil, nil, err
	}

	p.AddListener(NewListenLogger(logger, p))
	return &context, p, nil
}

func createVariableMap(stack *client.Stack, rancherCompose string) (map[string]interface{}, error) {
	variables := map[string]interface{}{}
	for k, v := range stack.Environment {
		variables[k] = v
	}

	// TODO
	/*questions, err := lookup.ParseQuestions([]byte(rancherCompose))
	if err != nil {
		return nil, err
	}

	for k, question := range questions {
		if _, ok := variables[k]; !ok {
			variables[k] = question.Default
		}
	}*/

	return variables, nil
}
