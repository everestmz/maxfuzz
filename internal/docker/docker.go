package docker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/everestmz/maxfuzz/internal/constants"

	d "github.com/fsouza/go-dockerclient"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/subosito/gotenv"
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
	portBindings  map[d.Port][]d.PortBinding
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
					Target:   constants.FuzzerOutputDirectory,
					Source:   c.syncDirectory,
					Type:     "bind",
					ReadOnly: false,
				},
			},
			AutoRemove:   true,
			PortBindings: c.portBindings,
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

	go followContainerCustomWriters(cont.ID, stdout, stderr)

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

func (c *FuzzCluster) Kill() error {
	var result *multierror.Error
	result = multierror.Append(result,
		client.StopContainer(c.Fuzzer, 1),
		client.RemoveContainer(
			d.RemoveContainerOptions{
				ID:    c.Fuzzer,
				Force: true,
			},
		),
	)

	return result.ErrorOrNil()
}

type FuzzClusterState struct {
	Fuzzer d.State
}

func (s *FuzzClusterState) Running() bool {
	return s.Fuzzer.Running
}

func (s *FuzzClusterState) ExitCode() int {
	return s.Fuzzer.ExitCode
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

func CreateFuzzer(target, baseImage string, stop chan bool, exposePorts map[string]string) (*FuzzClusterConfiguration, error) {
	toReturn := &FuzzClusterConfiguration{
		Target: target,
	}
	buildboxName := fmt.Sprintf("%s_buildbox", target)
	fuzzerName := fmt.Sprintf("%s_fuzzer", target)
	reproducerName := fmt.Sprintf("%s_reproducer", target)

	for container, host := range exposePorts {
		key := d.Port(fmt.Sprintf("%s/tcp", container))
		val := []d.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: host,
			},
		}
		toReturn.portBindings[key] = val
	}

	//Make sure all old containers are killed and removed (buildbox, fuzzer, repro)
	for _, box := range []string{buildboxName, fuzzerName, reproducerName} {
		client.StopContainer(box, 1)
		client.RemoveContainer(
			d.RemoveContainerOptions{
				ID:    box,
				Force: true,
			},
		)
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
		Image:        baseImage,
		AttachStdin:  true,
		AttachStdout: true,
		Entrypoint:   []string{constants.FuzzerBuildSteps},
		Env:          environment,
	}
	createContainerOptions := d.CreateContainerOptions{
		Name:   buildboxName,
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
					Target:   constants.FuzzerOutputDirectory,
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

	ticker := time.NewTicker(time.Second)
	for cont.State.Running {
		select {
		case <-stop:
			var result *multierror.Error
			result = multierror.Append(result,
				fmt.Errorf("Fuzzer creation stopped"),
				client.StopContainer(buildboxName, 1),
				client.RemoveContainer(
					d.RemoveContainerOptions{
						ID:    buildboxName,
						Force: true,
					},
				),
			)
			return nil, result.ErrorOrNil()
		case <-ticker.C:
			cont, err = client.InspectContainer(cont.ID)
			if err != nil {
				return nil, err
			}
		}
	}

	ticker.Stop()
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
