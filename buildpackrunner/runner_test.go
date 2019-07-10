package buildpackrunner_test

import (
	"code.cloudfoundry.org/buildpackapplifecycle"
	"code.cloudfoundry.org/buildpackapplifecycle/buildpackrunner"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)


// When (there is at least one launch.yml)

// When (there are NO launch.yml files)

var _ = Describe("Runner", func() {
	Context("StartCommand", func() {

		var runner *buildpackrunner.Runner
		var appDir string
		var buildpacks = []string{"haskell-buildpack", "bash-buildpack"}

		BeforeEach(func() {
			skipDetect := true
			builderConfig := buildpackapplifecycle.NewLifecycleBuilderConfig(buildpacks, skipDetect, false)
			for _, bp := range buildpacks {
				bpPath := builderConfig.BuildpackPath(bp)
				Expect(genFakeBuildpack(bpPath)).To(Succeed())
			}

			runner = buildpackrunner.New(&builderConfig)
			appDir = filepath.Join(os.TempDir(), "app")
			Expect(os.MkdirAll(appDir, os.ModePerm)).ToNot(HaveOccurred())
		})

		When("There is NO procfile and NO launch.yml file", func() {
			It("should use the default start command", func() {
				Expect(runner.Setup()).To(Succeed())

				resultsJSON, stagingInfo, err := runner.GoLikeLightning()

				Expect(err).NotTo(HaveOccurred())
				Expect(stagingInfo).To(ContainSubstring("staging_info.yml"))
				Expect(stagingInfo).To(BeAnExistingFile())

				stagingInfoContents, err := ioutil.ReadFile(stagingInfo)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stagingInfoContents)).To(ContainSubstring(`{"detected_buildpack":"","start_command":"I wish I was a baller"}`))

				resultsJSONContents, err := ioutil.ReadFile(resultsJSON)
				Expect(string(resultsJSONContents)).To(MatchJSON(`{
        "lifecycle_metadata": {
          "buildpack_key": "bash-buildpack",
          "detected_buildpack": "",
          "buildpacks": [
            {
              "key": "haskell-buildpack",
              "name": ""
            },
            {
              "key": "bash-buildpack",
              "name": ""
            }
          ]
        },
        "process_types": {
          "web": "I wish I was a baller"
        },
        "processes": [
          {
            "Type": "web",
            "Command": "I wish I was a baller"
          }
        ],
        "sidecars": null,
        "execution_metadata": "",
        "lifecycle_type": "buildpack"
				}`))
			})
		})

		When("A launch.yml is present and there is NO procfile", func() {
			It("Should use the start command from launch.yml", func() {
				Expect(runner.Setup()).To(Succeed())

				Expect(os.MkdirAll(runner.GetDepsDir(), os.ModePerm)).To(Succeed())
				defer os.RemoveAll(runner.GetDepsDir())

				launchContent := []string{`
processes:
- type: "web"
  command: "do something forever"
- type: "worker"
  command: "do something and then quit"
- type: "newrelic"
  command: "run new relic"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web" , "worker" ] `, `
processes:
- type: "web"
  command: "do something else forever"
- type: "oldrelic"
  command: "run new relic"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web" ] `}

				for index := range buildpacks {
					depsIdxPath := filepath.Join(runner.GetDepsDir(), strconv.Itoa(index))
					Expect(os.MkdirAll(depsIdxPath, os.ModePerm)).To(Succeed())
					launchPath := filepath.Join(depsIdxPath, "launch.yml")
					Expect(ioutil.WriteFile(launchPath, []byte(launchContent[index]), os.ModePerm)).To(Succeed())
				}

				resultsJSON, stagingInfo, err := runner.GoLikeLightning()

				Expect(err).NotTo(HaveOccurred())
				Expect(stagingInfo).To(ContainSubstring("staging_info.yml"))
				Expect(stagingInfo).To(BeAnExistingFile())

				stagingInfoContents, err := ioutil.ReadFile(stagingInfo)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stagingInfoContents)).To(ContainSubstring(`{"detected_buildpack":"","start_command":"do something else forever"}`))

				resultsJSONContents, err := ioutil.ReadFile(resultsJSON)
				Expect(string(resultsJSONContents)).To(MatchJSON(`{
        "lifecycle_metadata": {
          "buildpack_key": "bash-buildpack",
          "detected_buildpack": "",
          "buildpacks": [
            {
              "key": "haskell-buildpack",
              "name": ""
            },
            {
              "key": "bash-buildpack",
              "name": ""
            }
          ]
        },
        "process_types": {
          "newrelic": "run new relic",
          "oldrelic": "run new relic",
          "web": "do something else forever",
          "worker": "do something and then quit"
        },
        "processes": [
          {
            "Type": "web",
            "Command": "do something else forever"
          },
          {
            "Type": "worker",
            "Command": "do something and then quit"
          },
          {
            "Type": "newrelic",
            "Command": "run new relic"
          },
					{
            "Type": "oldrelic",
            "Command": "run new relic"
          }
        ],
        "sidecars": [
          {
            "Name": "newrelic",
            "ProcessTypes": [
              "web",
							"worker"
            ],
            "Command": "run new relic"
          },
					{
            "Name": "oldrelic",
            "ProcessTypes": [
              "web"
            ],
            "Command": "run new relic"
          }
        ],
        "execution_metadata": "",
        "lifecycle_type": "buildpack"
      }`))
			})
		})


		When("A procfile is present and there is NO launch.yml", func() {
			It("Should always use the start command from the procfile", func() {
				procFilePath := filepath.Join(appDir, "Procfile")
				Expect(ioutil.WriteFile(procFilePath, []byte("web: gunicorn server:app"), os.ModePerm)).To(Succeed())
				defer os.Remove(procFilePath)

				Expect(runner.Setup()).To(Succeed())
				_, output, err := runner.GoLikeLightning()

				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(ContainSubstring("staging_info.yml"))
				Expect(output).To(BeAnExistingFile())

				contents, err := ioutil.ReadFile(output)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(`{"detected_buildpack":"","start_command":"gunicorn server:app"}`))

			})
		})

		When("there is NO procfile present and there is launch.yml provided by supply buildpacks", func() {
			It("Should always use the start command from the bin/release", func() {
				Expect(runner.Setup()).To(Succeed())

				launchContents := `
processes:
- type: "web"
  command: "do something forever"
- type: "worker"
  command: "do something and then quit"
- type: "lightning"
  command: "go forth"
- type: "newrelic"
  command: "run new relic"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web" ]`


				depsIdxPath := filepath.Join(runner.GetDepsDir(), strconv.Itoa(0))
				Expect(os.MkdirAll(depsIdxPath, os.ModePerm)).To(Succeed())
				launchPath := filepath.Join(depsIdxPath, "launch.yml")
				Expect(ioutil.WriteFile(launchPath, []byte(launchContents), os.ModePerm)).To(Succeed())


				resultsJSON, stagingInfo, err := runner.GoLikeLightning()

				Expect(err).NotTo(HaveOccurred())
				Expect(stagingInfo).To(ContainSubstring("staging_info.yml"))
				Expect(stagingInfo).To(BeAnExistingFile())

				stagingInfoContents, err := ioutil.ReadFile(stagingInfo)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stagingInfoContents)).To(ContainSubstring(`{"detected_buildpack":"","start_command":"I wish I was a baller"}`))

				resultsJSONContents, err := ioutil.ReadFile(resultsJSON)
				Expect(string(resultsJSONContents)).To(MatchJSON(`{
        "lifecycle_metadata": {
          "buildpack_key": "bash-buildpack",
          "detected_buildpack": "",
          "buildpacks": [
            {
              "key": "haskell-buildpack",
              "name": ""
            },
            {
              "key": "bash-buildpack",
              "name": ""
            }
          ]
        },
        "process_types": {
          "lightning": "go forth",
          "newrelic": "run new relic",
          "web": "I wish I was a baller",
          "worker": "do something and then quit"
        },
        "processes": [
          {
            "Type": "web",
            "Command": "I wish I was a baller"
          },
          {
            "Type": "worker",
            "Command": "do something and then quit"
          },
          {
            "Type": "lightning",
            "Command": "go forth"
          },
          {
            "Type": "newrelic",
            "Command": "run new relic"
          }
        ],
        "sidecars": [
          {
            "Name": "newrelic",
            "ProcessTypes": [
              "web"
            ],
            "Command": "run new relic"
          }
        ],
        "execution_metadata": "",
        "lifecycle_type": "buildpack"
      }`))

			})
		})


		When("A procfile is present and there is launch.yml provided by all buildpacks", func() {
			It("Should always use the start command from the procfile", func() {
				procFilePath := filepath.Join(appDir, "Procfile")
				Expect(ioutil.WriteFile(procFilePath, []byte("web: gunicorn server:app"), os.ModePerm)).To(Succeed())
				defer os.Remove(procFilePath)

				Expect(runner.Setup()).To(Succeed())

				launchContent := []string{`
processes:
- type: "web"
  command: "do something forever"
- type: "worker"
  command: "do something and then quit"
- type: "lightning"
  command: "go forth"
- type: "newrelic"
  command: "run new relic"
  platforms:
    cloudfoundry:
      sidecar_for: [ "web" ] `, `
processes:
- type: "worker"
  command: "do something else forever"
- type: "oldrelic"
  command: "run new relic"
  platforms:
    cloudfoundry:
      sidecar_for: [ "worker" ] `}

				for index := range buildpacks {
					depsIdxPath := filepath.Join(runner.GetDepsDir(), strconv.Itoa(index))
					Expect(os.MkdirAll(depsIdxPath, os.ModePerm)).To(Succeed())
					launchPath := filepath.Join(depsIdxPath, "launch.yml")
					Expect(ioutil.WriteFile(launchPath, []byte(launchContent[index]), os.ModePerm)).To(Succeed())
				}

				resultsJSON, stagingInfo, err := runner.GoLikeLightning()

				Expect(err).NotTo(HaveOccurred())
				Expect(stagingInfo).To(ContainSubstring("staging_info.yml"))
				Expect(stagingInfo).To(BeAnExistingFile())

				stagingInfoContents, err := ioutil.ReadFile(stagingInfo)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stagingInfoContents)).To(ContainSubstring(`{"detected_buildpack":"","start_command":"gunicorn server:app"}`))

				resultsJSONContents, err := ioutil.ReadFile(resultsJSON)
				Expect(string(resultsJSONContents)).To(MatchJSON(`{
        "lifecycle_metadata": {
          "buildpack_key": "bash-buildpack",
          "detected_buildpack": "",
          "buildpacks": [
            {
              "key": "haskell-buildpack",
              "name": ""
            },
            {
              "key": "bash-buildpack",
              "name": ""
            }
          ]
        },
        "process_types": {
          "lightning": "go forth",
          "newrelic": "run new relic",
          "oldrelic": "run new relic",
          "web": "gunicorn server:app",
          "worker": "do something else forever"
        },
        "processes": [
          {
            "Type": "web",
            "Command": "gunicorn server:app"
          },
          {
            "Type": "worker",
            "Command": "do something else forever"
          },
          {
            "Type": "lightning",
            "Command": "go forth"
          },
          {
            "Type": "newrelic",
            "Command": "run new relic"
          },
          {
            "Type": "oldrelic",
            "Command": "run new relic"
          }
        ],
        "sidecars": [
          {
            "Name": "newrelic",
            "ProcessTypes": [
              "web"
            ],
            "Command": "run new relic"
          },
          {
            "Name": "oldrelic",
            "ProcessTypes": [
              "worker"
            ],
            "Command": "run new relic"
          }
        ],
        "execution_metadata": "",
        "lifecycle_type": "buildpack"
      }`))

			})
		})
	})
})

func genFakeBuildpack(bpRoot string) (error) {
	err := os.MkdirAll(filepath.Join(bpRoot, "bin"), os.ModePerm)
	if err != nil {
		return err
	}
	err = helper.CopyDirectory(filepath.Join("testdata", "bin"), filepath.Join(bpRoot, "bin"))
	if err != nil {
		return err
	}
	return nil
}
