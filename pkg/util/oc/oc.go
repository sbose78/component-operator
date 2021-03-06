/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package oc

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
)

var Client struct {
	Path string
	Pwd  string
}

type Command struct {
	Args   []string
	Data   *string
	Format string
}

func getClientPath() string {
	// Search for oc client
	ocpath, err := exec.LookPath("oc")
	if err != nil {
		log.Error("Can't find oc client")
	}
	return ocpath
}

func init() {
	Client.Path = getClientPath()
	Client.Pwd, _ = os.Getwd()
}

func ExecCommandAndReturn(command Command) (string, error) {
	cmd := exec.Command(Client.Path, command.Args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func ExecCommand(command Command) {
	cmd := exec.Command(Client.Path, command.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func Exists(kind string, name string) bool {
	s, err := ExecCommandAndReturn(Command{Args: []string{"get", kind + "/" + name, "-o", "jsonpath={.metadata.name}", "--ignore-not-found"}})
	if err != nil {
		return false
	} else {
		return name == s
	}
}

func GetNamesByLabel(kind string, labelName string, labelValue string) []string {
	s, err := ExecCommandAndReturn(Command{Args: []string{"get", kind, "-l", labelName + "=" + labelValue, "-o", "jsonpath={.items[*].metadata.name}"}})
	if err != nil {
		panic(err)
	} else if len(s) == 0 {
		return make([]string, 0)
	} else {
		return strings.Fields(s)
	}
}
