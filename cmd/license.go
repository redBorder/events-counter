// Simple utility for counting messages on a Kafka topic.
//
// Copyright (C) 2017 ENEO Tecnologia SL
// Author: Diego Fernández Barrera <bigomby@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/fatih/color"
)

// PubKey should be set on compile time
var PubKey string

type licenseStatus struct {
	OrganizationUUID string
	Amount           uint64
	Expired          bool
}

// LimitBytes is a map containing the max bytes for each organization
type LimitBytes map[string]*licenseStatus

func (l LimitBytes) getOrganizationLimits() map[string]uint64 {
	total := make(map[string]uint64)

	for _, v := range l {
		if !v.Expired {
			total[v.OrganizationUUID] += v.Amount
		}
	}

	return total
}

// License contains information about the limits of bytes that an organization
// can send.
type License struct {
	UUID         string         `json:"uuid"`
	ExpireAt     int64          `json:"expire_at"`
	LimitBytes   uint64         `json:"limit_bytes"`
	ClusterUUID  string         `json:"cluster_uuid"`
	Sensors      map[string]int `json:"sensors"`
	Organization string         `json:"organization_uuid"`
}

// LoadLicenses reads a redBorder license from a file and returns a License
// struct holding the decoded information.
func LoadLicenses(config *AppConfig) (LimitBytes, error) {
	limits := make(LimitBytes)

	log := log.WithField("prefix", "license")

	files, err := ioutil.ReadDir(config.LicensesDirectory)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		encodedLicense, encodedSignature, err :=
			ReadLicense(config.LicensesDirectory + "/" + file.Name())
		if err != nil {
			return nil, err
		}

		if PubKey != "" {
			pub, err := DecodePEM([]byte(PubKey))
			if err != nil {
				return nil, err
			}

			err = VerifyLicense(pub, encodedLicense, encodedSignature)
			if err != nil {
				log.Errorf(
					"Error verifying license \"%s\": %s", file.Name(), err.Error())
				continue
			}
		}

		license, err := DecodeLicense(encodedLicense)
		if err != nil {
			return nil, err
		}

		limits[license.UUID] = &licenseStatus{}

		expires := time.Unix(license.ExpireAt, 0)
		if expires.Before(time.Now()) {
			log.Warnf("License %s has expired", license.UUID)
			limits[license.UUID].Expired = true
			continue
		}

		log.Infoln(FormatLicense(license))

		if config.OrganizationMode {
			if license.Organization == "" {
				log.Warnf("Ignoring license WITH NO organization %s", license.UUID)
				continue
			}

			limits[license.UUID].Amount += license.LimitBytes
			limits[license.UUID].OrganizationUUID = license.Organization
		} else {
			if license.Organization != "" {
				log.Warnf("Ignoring license WITH organization %s", license.UUID)
				continue
			}

			limits[license.UUID].Amount += license.LimitBytes
			limits[license.UUID].OrganizationUUID = "*"
		}
	}

	return limits, nil
}

// ReadLicense read the license from a file. Decode as a base64 string and
// parses it as a JSON.
func ReadLicense(filename string) (encodedInfo, signature []byte, err error) {
	var (
		encodedSignatureStr string
		encodedInfoStr      string
		ok                  bool
	)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	var el map[string]interface{}
	err = json.Unmarshal(data, &el)
	if err != nil {
		return nil, nil, err
	}

	if encodedInfoStr, ok = el["encoded_info"].(string); !ok {
		return nil, nil, errors.New("Encoded info is not string")
	}

	if encodedSignatureStr, ok = el["signature"].(string); !ok {
		return nil, nil, errors.New("Signature info is not string")
	}

	return []byte(encodedInfoStr), []byte(encodedSignatureStr), nil
}

// DecodePEM reads and parses a PEM key with a public key
func DecodePEM(key []byte) (*rsa.PublicKey, error) {
	pemKey := strings.Replace(string(key), `\n`, "\n", -1)
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, errors.New("Error decoding PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("Key is not an RSA key")
	}

	return pubKey, nil
}

// VerifyLicense checks if a encoded license has a valid signature using a
// public key.
func VerifyLicense(
	pub *rsa.PublicKey, encodedLicense, encodedSignature []byte,
) error {
	hashed := sha256.Sum256(encodedLicense)
	signatureLen := base64.URLEncoding.DecodedLen(len(encodedSignature))
	signature := make([]byte, signatureLen)

	n, err := base64.URLEncoding.Decode(signature, encodedSignature)
	if err != nil {
		return err
	}

	err = rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], signature[:n])
	if err != nil {
		return err
	}

	return nil
}

// DecodeLicense receives a JSON marshaled license a returns a struct with the
// information.
func DecodeLicense(encodedLicense []byte) (*License, error) {
	license := new(License)

	dataLen := base64.URLEncoding.DecodedLen(len(encodedLicense))
	data := make([]byte, dataLen)

	n, err := base64.URLEncoding.Decode(data, encodedLicense)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data[:n], license)
	if err != nil {
		return nil, err
	}

	return license, nil
}

// FormatLicense create a pretty string given a license object
func FormatLicense(license *License) string {
	keyColor := color.New(color.FgYellow, color.Bold).SprintFunc()

	return fmt.Sprintf("Using license \n"+
		keyColor("\tUUID:              ")+"%s\n"+
		keyColor("\tCluster UUID:      ")+"%s\n"+
		keyColor("\tOrganization UUID: ")+"%s\n"+
		keyColor("\tExpires:           ")+"%s\n"+
		keyColor("\tLimit bytes:       ")+"%d",
		license.UUID,
		license.ClusterUUID,
		license.Organization,
		time.Unix(license.ExpireAt, 0).String(),
		license.LimitBytes,
	)
}
