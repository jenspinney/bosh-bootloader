package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/cloudfoundry/bosh-bootloader/fileio"
	uuid "github.com/nu7hatch/gouuid"
)

var (
	marshalIndent = json.MarshalIndent
	uuidNewV4     = uuid.NewV4
)

const (
	STATE_SCHEMA = 14

	OS_READ_WRITE_MODE = os.FileMode(0644)
	StateFileName      = "bbl-state.json"
)

type Store struct {
	dir         string
	fs          stateStoreFs
	stateSchema int
}

type stateStoreFs interface {
	fileio.FileWriter
	fileio.Remover
	fileio.AllRemover
	fileio.Stater
	fileio.AllMkdirer
}

func NewStore(dir string, fs stateStoreFs) Store {
	return Store{
		dir:         dir,
		fs:          fs,
		stateSchema: STATE_SCHEMA,
	}
}

func (s Store) Set(state State) error {
	_, err := s.fs.Stat(s.dir)
	if err != nil {
		return fmt.Errorf("Stat state dir: %s", err)
	}

	stateFile := filepath.Join(s.dir, StateFileName)
	if reflect.DeepEqual(state, State{}) {
		err := s.fs.Remove(stateFile)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		rmdir := func(getDirFunc func() (string, error)) error {
			d, _ := getDirFunc()
			return s.fs.RemoveAll(d)
		}
		if err := rmdir(s.GetCloudConfigDir); err != nil {
			return err
		}
		if err := rmdir(s.GetDirectorDeploymentDir); err != nil {
			return err
		}
		if err := rmdir(s.GetJumpboxDeploymentDir); err != nil {
			return err
		}
		if err := rmdir(s.GetVarsDir); err != nil {
			return err
		}
		if err := rmdir(s.GetTerraformDir); err != nil {
			return err
		}
		if err := rmdir(s.GetBblOpsFilesDir); err != nil {
			return err
		}

		_ = s.fs.Remove(filepath.Join(s.dir, "create-jumpbox.sh"))
		_ = s.fs.Remove(filepath.Join(s.dir, "create-director.sh"))
		_ = s.fs.Remove(filepath.Join(s.dir, "delete-jumpbox.sh"))
		_ = s.fs.Remove(filepath.Join(s.dir, "delete-director.sh"))

		return nil
	}

	state.Version = s.stateSchema

	if state.ID == "" {
		uuid, err := uuidNewV4()
		if err != nil {
			return fmt.Errorf("Create state ID: %s", err)
		}
		state.ID = uuid.String()
	}

	jsonData, err := marshalIndent(state, "", "\t")
	if err != nil {
		return err
	}
	err = s.fs.WriteFile(stateFile, jsonData, os.FileMode(0644))
	if err != nil {
		return err
	}

	return nil
}

func (s Store) GetStateDir() string {
	return s.dir
}

func (s Store) GetCloudConfigDir() (string, error) {
	return s.getDir("cloud-config")
}

func (s Store) GetTerraformDir() (string, error) {
	return s.getDir("terraform")
}

func (s Store) GetBblOpsFilesDir() (string, error) {
	return s.getDir("bbl-ops-files")
}

func (s Store) GetVarsDir() (string, error) {
	return s.getDir("vars")
}

func (s Store) GetDirectorDeploymentDir() (string, error) {
	return s.getDir("bosh-deployment")
}

func (s Store) GetJumpboxDeploymentDir() (string, error) {
	return s.getDir("jumpbox-deployment")
}

func (s Store) GetOldBblDir() string {
	return filepath.Join(s.dir, ".bbl")
}

func (s Store) getDir(name string) (string, error) {
	dir := filepath.Join(s.dir, name)
	err := s.fs.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return "", err
	}
	return dir, nil
}
