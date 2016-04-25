package main

import (
	"os"
	"path/filepath"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"model"
	"github.com/pborman/uuid"
	"strings"
	"github.com/correia-io/goutils/src/utils"
)

type Tortuga interface {
	Build()
	CheckHome()
	CheckProjectFile()
	Clean()
	Deploy2Nexus()
	Dist()
	GetProject()
	InstallGB()
	Release()
	RunTests()
}

type Turtle struct{
	Config  model.TurtleConfig
}

func (s Turtle) Build() {
	logger.Debug("Building go project @", TURTLE_PROJECT_PATH)
	rt.RunCommandLogStream("gb", []string{"build"})
	if _, err := os.Stat("dist"); os.IsNotExist(err) {
		os.Mkdir("dist", 00750)
	}
}

func (s Turtle) Clean() {
	fmt.Println("Cleaning the house")
	os.RemoveAll(TURTLE_PROJECT_PATH + "dist")
	os.RemoveAll(TURTLE_PROJECT_PATH + "pkg")
	os.RemoveAll(TURTLE_PROJECT_PATH + "bin")
	files, err := ioutil.ReadDir(".")
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "tar.gz") {
			fmt.Printf("Removing file:%s\n", file.Name())
			os.Remove(file.Name())
		}
	}
}

func (s Turtle) CheckHome() {
	if _, err := os.Stat(TURTLE_HOME); os.IsNotExist(err) {
		os.Mkdir(TURTLE_HOME, 00750)
	}
}

//func (s Turtle) CheckProjectFile() {
//	if _, err := os.Stat(TURTLE_FILE); os.IsNotExist(err) {
//		ct.Foreground(ct.Red, false)
//		resp := wiz.Question("Project doesn't have turtle.json file. Would you like to create one? [y/N] ")
//		project := model.Project{}
//		project.Version = "0.0.1-SNAPSHOT"
//		if strings.ToLower(resp) == "y" {
//			slice := strings.Split(dir, "/")
//			projectName := slice[len(slice) - 1]
//			ct.Foreground(ct.Cyan, false)
//			pName := wiz.QuestionF("Project Name: [%s] ", projectName)
//			if pName == "" {
//				if pName == "" {
//					project.Name = projectName
//				} else {
//					project.Name = pName
//				}
//			}
//
//			pGroup := wiz.QuestionF("GroupId: [%s] ", "com.company.my")
//			if pGroup == "" {
//				project.GroupId = "com.company.my"
//			} else {
//				project.GroupId = pGroup
//			}
//
//			pArti := wiz.QuestionF("ArtifactId: [%s] ", projectName)
//			if pArti == "" {
//				project.ArtifactId = projectName
//			} else {
//				project.ArtifactId = pArti
//			}
//
//			packaging := wiz.QuestionF("Packaging: [%s] ", "tar.gz")
//			if pArti == "" {
//				project.Packaging = "tar.gz"
//			} else {
//				project.Packaging = packaging
//			}
//
//			file, _ := json.MarshalIndent(&project, "", "  ")
//			wr := []byte(file)
//
//			err := ioutil.WriteFile(dir + "/turtle.json", wr, 0750)
//			if err != nil {
//				logger.Fatal(err)
//			}
//			ct.Foreground(ct.Yellow, false)
//			fmt.Println("Writing gobuilder config file...")
//			fmt.Println(string(wr))
//			ct.Foreground(ct.White, false)
//		} else {
//			fmt.Println("Aborted")
//			ct.Foreground(ct.White, false)
//			os.Exit(1)
//		}
//	}
//}

func (s Turtle) Dist() {
	if (project.ProjectType == "go") {
		goBuilder.Dist()
	} else if project.ProjectType == "static" {
		fmt.Println("Packaging Static Project")
		tmpDir := "/tmp/" + uuid.New()

		fmt.Println(os.Getwd())
		source, _ := os.Getwd()

		fileUtils.CopyDir(source, tmpDir + "/" + project.ArtifactId)

		os.RemoveAll("dist")
		if e, _ := fileUtils.Exists("dist"); !e {
			os.Mkdir("dist", 00750)
		}

		distName := fmt.Sprintf(source + "/dist/%s-%s.%s", project.ArtifactId, project.Version, project.Packaging)
		os.Chdir(tmpDir)
		fmt.Println(tmpDir)
		fmt.Println(distName)
		compressor.Tar(project.ArtifactId, distName)
		os.RemoveAll(tmpDir)
	}
}

func (s Turtle) InstallGB() {
	workdir := "/tmp/" + uuid.New()
	os.Chdir(workdir)
	os.Setenv("GOPATH", workdir)
	rt.RunCommandLogStream("go", []string{"get", "github.com/constabulary/gb/..."})
	fmt.Println("Copying GB binaries to /bin/")
	rt.RunCommandLogStream("sudo", []string{"cp", workdir + "/bin/gb", "/bin/gb"})
	rt.RunCommandLogStream("sudo", []string{"cp", workdir + "/bin/gb-vendor", "/bin/gbv-endor"})
	os.RemoveAll(workdir)
	fmt.Println("Done")
}

func (s Turtle) RunTests() {
	dir, err := filepath.Abs(filepath.Dir(TURTLE_PROJECT_PATH))
	os.Chdir(dir)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Debug("Runing tests @", dir)
	rt.RunCommandLogStream("gb", []string{"test"})
}

func (s Turtle) Deploy2Nexus(builds []string) {
	if !rt.CheckBinaryInPath("mvn") {
		logger.Fatal("Maven not found in PATH, please check your configuration.")
	} else {
		project := s.GetProject()
		var jobRepo model.Repository
		fmt.Println("Starting Deployment to Nexus Jobs:", builds)

		for _, build := range project.Builds {
			if utils.StringInSlice(build.ID, builds) {
				for _, rp := range s.Config.Repositories {
					if rp.Id == *deployToNexusRepId {
						jobRepo = rp
					}
				}

				// Overrides repositories from Turtle Config
				for _, rp := range project.Repositories {
					if rp.Id == *deployToNexusRepId {
						jobRepo = rp
					}
				}

				version := fmt.Sprintf("%s-%s-%s", build.OS, build.Arch, project.Version)
				file := fmt.Sprintf("%s-%s.%s", project.ArtifactId, version, project.Packaging)
				fmt.Println("Deploying file:", file)
				args := []string{
					"deploy:deploy-file",
					"-DgroupId=" + project.GroupId,
					"-DartifactId=" + project.ArtifactId,
					"-Dversion=" + version,
					"-Dpackaging=" + project.Packaging,
					"-Durl=" + jobRepo.URL,
					"-Dfile=" + file,
					"-DgeneratePom=" + *deployToNexusGeneratePom,
					"-DrepositoryId=" + jobRepo.Id,
				}
				err := rt.RunCommandLogStream("mvn", args)
				if err != nil {
					logger.Fatal(err)
				}
			}
		}

	}
}

func (s Turtle) GetProject() (model.Project) {
	projectFile, err := ioutil.ReadFile(TURTLE_FILE)
	var project model.Project
	if err != nil {
		logger.Error("Workspace busted", err, TURTLE_FILE)
	} else {
		var c model.Project
		err := json.Unmarshal(projectFile, &c)
		if err != nil {
			logger.Error(err)
		}
		project = c
	}

	return project
}

func (s Turtle) Release() {
	fmt.Println("Releasing project", app.Name, "-", TURTLE_VERSION)
}
