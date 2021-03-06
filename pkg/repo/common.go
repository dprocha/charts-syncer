package repo

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
	"github.com/juju/errors"
	"k8s.io/klog"
)

// download downloads a packaged from the given repo.
func download(filepath string, downloadURL string, sourceRepo *api.Repo) error {
	// Get the data
	req, err := http.NewRequest("GET", downloadURL, nil)
	klog.V(4).Infof("GET %q", downloadURL)
	if err != nil {
		return errors.Annotatef(err, "error getting chart from %q", downloadURL)
	}
	if sourceRepo.Auth != nil && sourceRepo.Auth.Username != "" && sourceRepo.Auth.Password != "" {
		klog.V(4).Infof("Using basic authentication %q:****", sourceRepo.Auth.Username)
		req.SetBasicAuth(sourceRepo.Auth.Username, sourceRepo.Auth.Password)
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotate(err, "error doing request")
	}
	defer res.Body.Close()

	// Check status code
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return errors.Errorf("error downloading chart %s. Status code is %d", downloadURL, res.StatusCode)
	}
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return errors.Annotatef(err, "error creating %s file", filepath)
	}
	defer out.Close()

	// Write the body to file
	if _, err = io.Copy(out, res.Body); err != nil {
		return errors.Annotatef(err, "error write to file %s", filepath)
	}

	return errors.Trace(err)
}

// pushToChartMuseumLike pushes a chart to a repo implementing the ChartMuseum API (chartmuseum and harbor)
func pushToChartMuseumLike(apiEndpoint string, filepath string, targetRepo *api.Repo) error {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("chart", filepath)
	if err != nil {
		return errors.Annotate(err, "error writing to buffer")
	}

	fh, err := os.Open(filepath)
	if err != nil {
		return errors.Annotatef(err, "error opening file %s", filepath)
	}
	defer fh.Close()

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return errors.Trace(err)
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	req, err := http.NewRequest("POST", apiEndpoint, bodyBuf)
	klog.V(4).Infof("POST %q", apiEndpoint)
	req.Header.Add("content-type", contentType)
	if err != nil {
		return errors.Annotatef(err, "error creating POST request to %s", apiEndpoint)
	}
	if targetRepo.Auth != nil && targetRepo.Auth.Username != "" && targetRepo.Auth.Password != "" {
		klog.V(4).Info("Target repo uses basic authentication...")
		req.SetBasicAuth(targetRepo.Auth.Username, targetRepo.Auth.Password)
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotatef(err, "error doing POST request to %s", apiEndpoint)
	}
	defer res.Body.Close()
	respBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Annotatef(err, "error reading POST response from %s", apiEndpoint)
	}
	klog.V(4).Infof("POST chart status Code: %d, Message: %s", res.StatusCode, string(respBody))
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		klog.V(3).Infof("Chart %s uploaded successfully", filepath)
	} else {
		return errors.Errorf("POST chart status Code: %d, Message: %s", res.StatusCode, string(respBody))
	}
	return errors.Trace(err)
}

// downloadFromChartMuseumLike downloads a chart from a repo implementing the ChartMuseum API (chartmuseum and harbor)
func downloadFromChartMuseumLike(apiEndpoint string, filepath string, sourceRepo *api.Repo) error {
	if err := download(filepath, apiEndpoint, sourceRepo); err != nil {
		return errors.Trace(err)
	}
	// Check contentType
	contentType, err := utils.GetFileContentType(filepath)
	if err != nil {
		return errors.Trace(err)
	}
	if contentType != "application/x-gzip" {
		return errors.Errorf("the downloaded chart %s is not a gzipped tarball", filepath)
	}
	return nil
}
