package e2e

import (
	"archive/zip"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pkg/errors"
	v1alpha1 "github.com/snowdrop/component-operator/pkg/apis/component/v1alpha1"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"strings"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	// Register the core k8s types
	cgoscheme.AddToScheme(scheme)
	// Register custom resource type
	v1alpha1.SchemeBuilder.AddToScheme(scheme)
}


func runCmd(cmdS string) string {
	cmd := exec.Command("/bin/sh", "-c", cmdS)
	fmt.Fprintf(GinkgoWriter, "Running command: %s\n", cmdS)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

	// wait for the command execution to complete
	<-session.Exited
	Expect(session.ExitCode()).To(Equal(0))
	Expect(err).NotTo(HaveOccurred())

	return string(session.Out.Contents())
}

func crudClient() client.Client {
	kubeconfig, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	runtimeClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		panic(err)
	}
	return runtimeClient
}

func springBootComponent(name, ns string) *v1alpha1.Component {
	return &v1alpha1.Component{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind: v1alpha1.ComponentKind,
		},
		ObjectMeta: metav1.ObjectMeta{
		  	Name: name,
		  	Namespace: ns,
		},
		Spec: v1alpha1.ComponentSpec{
			DeploymentMode: "innerloop",
			Runtime: "spring-boot",
		},
	}
}

func generateSpringBootJavaProject(outDir, template, artifactid string) {
	serviceEndpoint := "http://spring-boot-generator.195.201.87.126.nip.io"

	client := http.Client{}
	form := url.Values{}
	form.Add("artifactid", artifactid)
	form.Add("module",template)
	form.Add("packagename", "org.acme")
	parameters := form.Encode()
	if parameters != "" {
		parameters = "?" + parameters
	}

	u := strings.Join([]string{serviceEndpoint, "app"}, "/") + parameters
	log.Infof("URL of the request calling the service is %s", u)
	req, err := http.NewRequest(http.MethodGet, u, strings.NewReader(""))

	if err != nil {
		log.Error(err.Error())
	}
	addClientHeader(req)

	res, err := client.Do(req)
	if err != nil {
		log.Error(err.Error())
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(err.Error())
	}

	zipFile := outDir + ".zip"

	err = ioutil.WriteFile(zipFile, body, 0644)
	if err != nil {
		log.Errorf("Failed to download file %s due to %s", zipFile, err)
	}
	err = Unzip(zipFile, outDir)
	if err != nil {
		log.Errorf("Failed to unzip new project file %s due to %s", zipFile, err)
	}
	err = os.Remove(zipFile)
	if err != nil {
		log.Errorf(err.Error())
	}
}

func addClientHeader(req *http.Request) {
	userAgent := "sd/1.0"
	req.Header.Set("User-Agent", userAgent)
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		name := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(name, os.ModePerm)
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(name, string(os.PathSeparator)); lastIndex > -1 {
				fdir = name[:lastIndex]
			}

			err = os.MkdirAll(fdir, os.ModePerm)
			if err != nil {
				return err
			}
			f, err := os.OpenFile(
				name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func runMavenBuild(buildDir string) error {
	goal := fmt.Sprintf("package")
	cmd := exec.Command("mvn", mavenExtraOptions(buildDir), goal)
	cmd.Dir = buildDir
	//cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Infof("running maven package: %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failure while determining classpath")
	}

	log.Info("Maven build completed successfully")
	return nil
}

func mavenExtraOptions(tmpDir string) string {
	return "-Dmaven.repo.local=" + tmpDir + "/m2"
}
