package docker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/subosito/gotenv"

	"github.com/everestmz/maxfuzz/internal/constants"
	d "github.com/fsouza/go-dockerclient"
)

var client *d.Client

func Init() error {
	var err error

	endpoint := "unix:///var/run/docker.sock"
	client, err = d.NewClient(endpoint)
	return err
}

type FuzzClusterConfiguration struct {
	Target        string //target id
	imageID       string //base image with built fuzzer
	environment   []string
	syncDirectory string
}

func (c *FuzzClusterConfiguration) Deploy(command []string, stdout, stderr io.Writer) (*FuzzCluster, error) {
	configuration := d.Config{
		Image:        c.imageID,
		AttachStdin:  true,
		AttachStdout: true,
		Entrypoint:   command,
		Env:          c.environment,
	}
	createContainerOptions := d.CreateContainerOptions{
		Name:   fmt.Sprintf("%s_fuzzer", c.Target),
		Config: &configuration,
		HostConfig: &d.HostConfig{
			Mounts: []d.HostMount{
				{
					Target:   constants.FuzzerLocation,
					Source:   filepath.Join(constants.LocalTargetDirectory, c.Target),
					Type:     "bind",
					ReadOnly: false,
				},
				{
					Target:   constants.FuzzerSyncDirectory,
					Source:   c.syncDirectory,
					Type:     "bind",
					ReadOnly: false,
				},
			},
		},
		NetworkingConfig: &d.NetworkingConfig{},
	}

	cont, err := client.CreateContainer(createContainerOptions)
	if err != nil {
		return nil, err
	}

	err = client.StartContainer(cont.ID, &d.HostConfig{})
	if err != nil {
		return nil, err
	}

	err = followContainer(cont.ID)
	if err != nil {
		return nil, err
	}

	return &FuzzCluster{
		Target:        c.Target,
		Fuzzer:        cont.Name,
		imageID:       c.imageID,
		environment:   c.environment,
		syncDirectory: c.syncDirectory,
	}, nil
}

type FuzzCluster struct {
	Target        string //target id
	Fuzzer        string
	imageID       string //base image with built fuzzer
	environment   []string
	syncDirectory string
}

func (c *FuzzCluster) State() (*FuzzClusterState, error) {
	toReturn := &FuzzClusterState{}
	fuzzer, err := client.InspectContainer(c.Fuzzer)
	if err != nil {
		return toReturn, err
	}
	toReturn.Fuzzer = fuzzer.State
	return toReturn, nil
}

type FuzzClusterState struct {
	Fuzzer d.State
}

func (s *FuzzClusterState) Running() bool {
	return s.Fuzzer.Running
}

func followContainer(id string) error {
	options := d.LogsOptions{
		Container:    id,
		Since:        0,
		Stdout:       true,
		Stderr:       true,
		Follow:       true,
		OutputStream: stdoutWriter{},
		ErrorStream:  stderrWriter{},
	}

	return client.Logs(options)
}

func followContainerCustomWriters(id string, out, err io.Writer) error {
	options := d.LogsOptions{
		Container:    id,
		Since:        0,
		Stdout:       true,
		Stderr:       true,
		Follow:       true,
		OutputStream: out,
		ErrorStream:  err,
	}

	return client.Logs(options)
}

func targetToRepository(target string) string {
	return fmt.Sprintf("%s_images", target)
}

func CreateFuzzer(target string, stop chan bool) (*FuzzClusterConfiguration, error) {
	toReturn := &FuzzClusterConfiguration{
		Target: target,
	}

	environmentFile, err := os.Open(filepath.Join(constants.LocalTargetDirectory, target, "environment"))
	if err != nil {
		return nil, err
	}
	environmentMap := gotenv.Parse(environmentFile)
	environment := []string{}
	for k, v := range environmentMap {
		environment = append(environment, fmt.Sprintf("%s=%s", k, v))
	}
	environmentFile.Close()

	// Ensure sync dir exists
	syncDirectory := filepath.Join(constants.LocalSyncDirectory, target)
	err = os.MkdirAll(syncDirectory, 0755)
	if err != nil {
		return nil, err
	}

	configuration := d.Config{
		Image:        "maxfuzz",
		AttachStdin:  true,
		AttachStdout: true,
		Entrypoint:   []string{constants.FuzzerBuildSteps},
		Env:          environment,
	}
	createContainerOptions := d.CreateContainerOptions{
		Name:   fmt.Sprintf("%s_buildbox", target),
		Config: &configuration,
		HostConfig: &d.HostConfig{
			Mounts: []d.HostMount{
				{
					Target:   constants.FuzzerLocation,
					Source:   filepath.Join(constants.LocalTargetDirectory, target),
					Type:     "bind",
					ReadOnly: false,
				},
				{
					Target:   constants.FuzzerSyncDirectory,
					Source:   syncDirectory,
					Type:     "bind",
					ReadOnly: false,
				},
			},
		},
		NetworkingConfig: &d.NetworkingConfig{},
	}

	cont, err := client.CreateContainer(createContainerOptions)
	if err != nil {
		return nil, err
	}

	err = client.StartContainer(cont.ID, &d.HostConfig{})
	if err != nil {
		return nil, err
	}

	go followContainer(cont.ID)

	cont, err = client.InspectContainer(cont.ID)
	if err != nil {
		return nil, err
	}

	for cont.State.Running {
		cont, err = client.InspectContainer(cont.ID)
		if err != nil {
			return nil, err
		}
		if len(stop) > 0 {
			<-stop
			err = client.StopContainer(cont.ID, 1)
			err = client.RemoveContainer(
				d.RemoveContainerOptions{
					ID:    cont.ID,
					Force: true,
				},
			)
			return nil, err
		}
	}

	if cont.State.Status != "FINISHED" && cont.State.ExitCode != 0 {
		return nil, fmt.Errorf("Error running build files - please check logs")
	}

	image, err := client.CommitContainer(
		d.CommitContainerOptions{
			Container:  cont.ID,
			Repository: targetToRepository(target),
		},
	)
	if err != nil {
		return nil, err
	}

	err = client.RemoveContainer(
		d.RemoveContainerOptions{
			ID:    cont.ID,
			Force: true,
		},
	)
	if err != nil {
		return nil, err
	}

	toReturn.imageID = image.ID
	toReturn.environment = environment
	toReturn.syncDirectory = syncDirectory
	return toReturn, err
}
