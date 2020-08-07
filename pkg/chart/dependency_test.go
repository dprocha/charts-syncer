package chart

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
	"gopkg.in/yaml.v2"
	helmChart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

func TestSyncDependencies(t *testing.T) {
	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)

	sourceChart := "../../testdata/kafka-10.3.3.tgz"
	if err := utils.Untar(sourceChart, testTmpDir); err != nil {
		t.Fatal(err)
	}

	sourceIndex, err := utils.LoadIndexFromRepo(source.Repo)
	if err != nil {
		t.Fatalf("error loading index.yaml: %v", err)
	}
	targetIndex := repo.NewIndexFile()

	chartPath := path.Join(testTmpDir, "kafka")
	err = syncDependencies(chartPath, source.Repo, target, sourceIndex, targetIndex, false)
	expectedError := "please sync zookeeper-5.14.3 dependency first"
	if err != nil && err.Error() != expectedError {
		t.Errorf("incorrect error, got: \n %s \n, want: \n %s \n", err.Error(), expectedError)
	}
}

func TestUpdateRequirementsFile(t *testing.T) {
	lock := &helmChart.Lock{
		Generated: time.Now(),
		Digest:    "sha256:fe26de7fc873dc8001404168feb920a61ba884a2fe211a7371165ed51bf8cb8b",
		Dependencies: []*helmChart.Dependency{
			{Name: "zookeeper", Version: "5.5.5"},
		},
	}

	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)

	sourceChart := "../../testdata/kafka-10.3.3.tgz"
	if err := utils.Untar(sourceChart, testTmpDir); err != nil {
		t.Fatal(err)
	}

	chartPath := path.Join(testTmpDir, "kafka")
	requirementsFile := path.Join(chartPath, "requirements.yaml")
	if err := updateRequirementsFile(chartPath, lock, source.Repo, target); err != nil {
		t.Fatal(err)
	}

	requirements, err := ioutil.ReadFile(requirementsFile)
	if err != nil {
		t.Fatalf("error reading updated %s file", requirementsFile)
	}

	newDeps := &dependencies{}
	err = yaml.Unmarshal(requirements, newDeps)
	if err != nil {
		t.Fatalf("error unmarshaling %s file", requirementsFile)
	}

	if newDeps.Dependencies[0].Repository != target.Repo.Url {
		t.Errorf("incorrect modification, got: %s, want: %s", newDeps.Dependencies[0].Repository, target.Repo.Url)
	}
	if newDeps.Dependencies[0].Version != "5.5.5" {
		t.Errorf("incorrect modification, got: %s, want: %s", newDeps.Dependencies[0].Version, "5.5.5")
	}
}

func TestWriteRequirementsFile(t *testing.T) {
	target := &api.TargetRepo{
		Repo: &api.Repo{
			Url:  "http://fake.target/com",
			Kind: api.Kind_CHARTMUSEUM,
			Auth: &api.Auth{
				Username: "user",
				Password: "password",
			},
		},
		ContainerRegistry:   "test.registry.io",
		ContainerRepository: "test/repo",
	}

	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)

	sourceChart := "../../testdata/kafka-10.3.3.tgz"
	if err := utils.Untar(sourceChart, testTmpDir); err != nil {
		t.Fatal(err)
	}

	chartPath := path.Join(testTmpDir, "kafka")
	requirementsFile := path.Join(chartPath, "requirements.yaml")
	requirements, err := ioutil.ReadFile(requirementsFile)
	if err != nil {
		t.Fatalf("error reading %s file", requirementsFile)
	}

	deps := &dependencies{}
	err = yaml.Unmarshal(requirements, deps)
	if err != nil {
		t.Fatalf("error unmarshaling %s file", requirementsFile)
	}

	deps.Dependencies[0].Repository = target.Repo.Url

	if err := writeRequirementsFile(chartPath, deps); err != nil {
		t.Fatal(err)
	}

	requirements, err = ioutil.ReadFile(requirementsFile)
	if err != nil {
		t.Fatalf("error reading updated %s file", requirementsFile)
	}

	newDeps := &dependencies{}
	err = yaml.Unmarshal(requirements, newDeps)
	if err != nil {
		t.Fatalf("error unmarshaling %s file", requirementsFile)
	}

	if newDeps.Dependencies[0].Repository != target.Repo.Url {
		t.Errorf("incorrect modification, got: %s, want: %s", newDeps.Dependencies[0].Repository, target.Repo.Url)
	}
	if newDeps.Dependencies[0].Version != "5.x.x" {
		t.Errorf("incorrect modification, got: %s, want: %s", newDeps.Dependencies[0].Version, "5.x.x")
	}
}

func TestFindDepByName(t *testing.T) {
	deps := &dependencies{
		Dependencies: []*helmChart.Dependency{
			{Name: "mariadb", Version: "1.2.3"},
			{Name: "postgresql", Version: "4.5.6"},
		},
	}
	dep := findDepByName(deps.Dependencies, "postgresql")
	if dep.Name != "postgresql" {
		t.Errorf("wrong dependency, got: %s , want: %s", dep.Name, "postgresql")
	}
	if dep.Version != "4.5.6" {
		t.Errorf("wrong dependency, got: %s , want: %s", dep.Version, "4.5.6")
	}
}
