/*
Copyright (C) 2018 Synopsys, Inc.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements. See the NOTICE file
distributed with this work for additional information
regarding copyright ownership. The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the License for the
specific language governing permissions and limitations
under the License.
*/

package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/blackducksoftware/perceptor-scanner/pkg/api"
	"github.com/blackducksoftware/perceptor-scanner/pkg/common"
	"github.com/prometheus/common/log"
)

const (
	imageFacadePort = "3004"
	pullImagePath   = "pullimage"
	checkImagePath  = "checkimage"
	baseURL         = "http://localhost"
)

type imageFacadePuller struct {
	httpClient *http.Client
}

func newImageFacadePuller() *imageFacadePuller {
	return &imageFacadePuller{httpClient: &http.Client{Timeout: 5 * time.Second}}
}

func (ifp *imageFacadePuller) PullImage(image *common.Image) error {
	log.Infof("attempting to pull image %s", image.PullSpec)

	err := ifp.startImagePull(image)
	if err != nil {
		log.Errorf("unable to pull image %s: %s", image.PullSpec, err.Error())
		return err
	}

	for {
		time.Sleep(5 * time.Second)
		var isDone bool
		isDone, err = ifp.checkImage(image)

		// TODO add some better error handling
		if err != nil {
			log.Errorf("unable to check image %s: %s", image.PullSpec, err.Error())
		}

		if isDone {
			log.Infof("finished pulling image %s", image.PullSpec)
			break
		}
	}

	return nil
}

func (ifp *imageFacadePuller) startImagePull(image *common.Image) error {
	url := fmt.Sprintf("%s:%s/%s", baseURL, imageFacadePort, pullImagePath)

	requestBytes, err := json.Marshal(image)
	if err != nil {
		log.Errorf("unable to marshal JSON for %s: %s", image.PullSpec, err.Error())
		return err
	}

	resp, err := ifp.httpClient.Post(url, "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		log.Errorf("unable to create request to %s for image %s: %s", url, image.PullSpec, err.Error())
		return err
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("image pull to %s failed with status code %d", url, resp.StatusCode)
		log.Errorf(err.Error())
		return err
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("unable to read body for image %s: %s", image.PullSpec, err.Error())
		return err
	}

	var pullImage api.PullImageResponse
	err = json.Unmarshal(bodyBytes, &pullImage)
	if err != nil {
		log.Errorf("unable to unmarshal body for image %s: %s", image.PullSpec, err.Error())
		return err
	}

	if !pullImage.IsSuccess {
		err = fmt.Errorf("image pull %s failed", image.PullSpec)
		log.Errorf(err.Error())
		return err
	}

	log.Infof("image pull for image %s succeeded", image.PullSpec)

	return nil
}

func (ifp *imageFacadePuller) checkImage(image *common.Image) (bool, error) {
	url := fmt.Sprintf("%s:%s/%s?", baseURL, imageFacadePort, checkImagePath)

	requestBytes, err := json.Marshal(image)
	if err != nil {
		log.Errorf("unable to marshal JSON for %s: %s", image.PullSpec, err.Error())
		return false, err
	}

	resp, err := ifp.httpClient.Post(url, "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		log.Errorf("unable to create request to %s for image %s: %s", url, image.PullSpec, err.Error())
		return false, err
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("GET %s failed with status code %d", url, resp.StatusCode)
		log.Errorf(err.Error())
		return false, err
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		recordError("unable to read response body")
		log.Errorf("unable to read response body from %s: %s", url, err.Error())
		return false, err
	}

	var getImage api.CheckImageResponse
	err = json.Unmarshal(bodyBytes, &getImage)
	if err != nil {
		recordError("unmarshaling JSON body failed")
		log.Errorf("unmarshaling JSON body bytes %s failed for URL %s: %s", string(bodyBytes), url, err.Error())
		return false, err
	}

	log.Infof("image check for image %s succeeded", image.PullSpec)

	return getImage.IsDone, nil
}
