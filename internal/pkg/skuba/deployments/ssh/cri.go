/*
 * Copyright (c) 2019 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package ssh

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"

	"github.com/SUSE/skuba/pkg/skuba"
)

func init() {
	stateMap["cri.configure"] = criConfigure
	stateMap["cri.sysconfig"] = criSysconfig
	stateMap["cri.start"] = criStart
	stateMap["crio16to18upgrade"] = crioUpgrade16to18
}

func criConfigure(t *Target, data interface{}) error {
	criFiles, err := ioutil.ReadDir(skuba.CriConfDir())
	if err != nil {
		return errors.Wrap(err, "Could not read local cri directory: "+skuba.CriConfDir())
	}
	defer func() {
		_, _, err := t.ssh("rm -rf /tmp/crio.conf.d")
		if err != nil {
			// If the deferred function has any return values, they are discarded when the function completes
			// https://golang.org/ref/spec#Defer_statements
			fmt.Println("Could not delete the crio.conf.d config path")
		}
	}()

	for _, f := range criFiles {
		if f.Name() != "README" {
			if err := t.target.UploadFile(filepath.Join(skuba.CriConfDir(), f.Name()), filepath.Join("/tmp/crio.conf.d", f.Name())); err != nil {
				return err
			}
		}
	}

	_, _, err = t.ssh("mv -f /tmp/crio.conf.d/01-caasp.conf /etc/crio/crio.conf.d/01-caasp.conf")
	return err
}

func criSysconfig(t *Target, data interface{}) error {
	criFiles, err := ioutil.ReadDir(skuba.CriDir())
	if err != nil {
		return errors.Wrap(err, "Could not read local cri directory: "+skuba.CriDir())
	}
	defer func() {
		_, _, err := t.ssh("rm -rf /tmp/cri.d")
		if err != nil {
			// If the deferred function has any return values, they are discarded when the function completes
			// https://golang.org/ref/spec#Defer_statements
			fmt.Println("Could not delete the cri.d config path")
		}
	}()

	for _, f := range criFiles {
		if err := t.target.UploadFile(filepath.Join(skuba.CriDir(), f.Name()), filepath.Join("/tmp/cri.d", f.Name())); err != nil {
			return err
		}
	}

	if _, _, err = t.ssh("mv -f /etc/sysconfig/crio /etc/sysconfig/crio.backup"); err != nil {
		return err
	}
	_, _, err = t.ssh("mv -f /tmp/cri.d/default_flags /etc/sysconfig/crio")
	return err
}

func criStart(t *Target, data interface{}) error {
	_, _, err := t.ssh("systemctl", "enable", "--now", "crio")
	return err
}

func crioUpgrade16to18(t *Target, data interface{}) error {
	_, _, err := t.ssh("mv /etc/sysconfig/crio /etc/sysconfig/crio.116")

	fileloc := "/etc/crio/crio.conf.d/01-caasp.conf"
	filecontent := `# This should not be edited, and are the defaults provided by CaaSP
	{{if not .StrictCapDefaults}}[crio.runtime]

	default_capabilities = [
		"CHOWN",
		"DAC_OVERRIDE",
		"FSETID",
		"FOWNER",
		"NET_RAW",
		"SETGID",
		"SETUID",
		"SETPCAP",
		"NET_BIND_SERVICE",
		"SYS_CHROOT",
		"KILL",
		"MKNOD",
		"AUDIT_WRITE",
		"SETFCAP",
	]{{end}}

	[crio.image]

	pause_image = "{{.PauseImage}}"
	`

	f, err := os.Create(fileloc)
	if err != nil {
		return errors.Wrapf(err, "could not create file %q", fileloc)
	}
	str, err := renderTemplate(filecontent, data)
	if err != nil {
		return errors.Wrap(err, "unable to render template")
	}
	_, err = f.WriteString(str)
	if err != nil {
		return errors.Wrapf(err, "unable to write template to file %s", f.Name())
	}
	if err := f.Chmod(0600); err != nil {
		return errors.Wrapf(err, "unable to chmod file %s", f.Name())
	}
	if err := f.Close(); err != nil {
		return errors.Wrapf(err, "unable to close file %s", f.Name())
	}
}

func renderTemplate(templateContents string, data interface{}) (string, error) {
	template, err := template.New("").Parse(templateContents)
	if err != nil {
		return "", errors.Wrap(err, "could not parse template")
	}
	var rendered bytes.Buffer
	if err := template.Execute(&rendered, data); err != nil {
		return "", errors.Wrap(err, "could not render configuration")
	}
	return rendered.String(), nil
}
