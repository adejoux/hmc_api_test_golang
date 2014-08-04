//author: adejoux@fr.ibm.com
// description: basic test of HMC REST api. Authenticate and get logical partition informations

package main

import (
  "fmt"
  "flag"
  "text/template"
  "net/http"
  "net/http/cookiejar"
  "crypto/tls"
  "io/ioutil"
  "log"
  "bytes"
  "encoding/xml"
)

//
// XML parsing structures
//

type Feed struct {
  XMLName   xml.Name   `xml:"feed"`
  Entries []Entry `xml:"entry"`
}
type Entry struct {
  XMLName   xml.Name   `xml:"entry"`
  Contents []Content `xml:"content"`
}

type Content struct {
  XMLName xml.Name `xml:"content"`
  Lpar []LogicalPartition `xml:"http://www.ibm.com/xmlns/systems/power/firmware/uom/mc/2012_10/ LogicalPartition"`
}

type LogicalPartition struct {
  XMLName xml.Name `xml:"http://www.ibm.com/xmlns/systems/power/firmware/uom/mc/2012_10/ LogicalPartition"`
  PartitionName string
  PartitionID int
  PartitionUUID string
  LogicalSerialNumber string
  OperatingSystemVersion string
}

//
// HTTP session struct
//

type Session struct {
  client *http.Client
  User string
  Password string
  url string
}

func NewSession(user string, password string, url string) *Session {
  tr := &http.Transport{
    TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
  }

  jar, err := cookiejar.New(nil)
    if err != nil {
        log.Fatal(err)
  }

  return &Session{ client : &http.Client{Transport: tr, Jar: jar}, User: user, Password: password, url: url }
}

func (s *Session) doLogon() {

  authurl := s.url + "/rest/api/web/Logon"

  // template for login request
  logintemplate := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
  <LogonRequest xmlns="http://www.ibm.com/xmlns/systems/power/firmware/web/mc/2012_10/" schemaVersion="V1_1_0">
    <Metadata>
      <Atom/>
    </Metadata>
    <UserID kb="CUR" kxe="false">{{.User}}</UserID>
    <Password kb="CUR" kxe="false">{{.Password}}</Password>
  </LogonRequest>`

  tmpl := template.New("logintemplate")
  tmpl.Parse(logintemplate)
  authrequest := new(bytes.Buffer)
  err := tmpl.Execute(authrequest, s)
  if err != nil {
    log.Fatal(err)
  }

  request, err := http.NewRequest("PUT", authurl, authrequest)

  // set request headers
  request.Header.Set("Content-Type", "application/vnd.ibm.powervm.web+xml; type=LogonRequest")
  request.Header.Set("Accept", "application/vnd.ibm.powervm.web+xml; type=LogonResponse")
  request.Header.Set("X-Audit-Memento", "hmctest")

  response, err := s.client.Do(request)
  if err != nil {
    log.Fatal(err)
  } else {
    defer response.Body.Close()
    if response.StatusCode != 200 {
      log.Fatalf("Error status code: %d", response.StatusCode)
    }
  }
}

func (s *Session) getManaged() {
  mgdurl := s.url + "/rest/api/uom/LogicalPartition"
  request, err := http.NewRequest("GET", mgdurl, nil)

  request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

  response, err := s.client.Do(request)
  if err != nil {
    log.Fatal(err)
  } else {
    defer response.Body.Close()
    contents, err := ioutil.ReadAll(response.Body)
    if err != nil {
      log.Fatal(err)
    }

    if response.StatusCode != 200 {
      log.Fatalf("Error getting LPAR informations. status code: %d", response.StatusCode)
    }

    var feed Feed
    new_err := xml.Unmarshal(contents, &feed)

    if new_err != nil {
      log.Fatal(new_err)
    }

    fmt.Printf("\t%-10s\t%-40s\t%-8s\t%-25s\n", "partition", "UUID", "LSERIAL", "OS" )
    for _, entry := range feed.Entries {
      for _, content := range entry.Contents {
        for _, lpar := range content.Lpar {
          fmt.Printf("\t%-10s\t%-40s\t%-8s\t%-25s\n", lpar.PartitionName, lpar.PartitionUUID, lpar.LogicalSerialNumber, lpar.OperatingSystemVersion)
        }
      }
    }
  }
}

//
// MAIN
//

func main() {

  user := flag.String("user","hscroot", "hmc user")
  password :=  flag.String("password","abc123", "hmc user password")
  url :=  flag.String("url","https://myhmc:12443", "hmc REST api url")

  flag.Parse()

  //initialize new http session
  session := NewSession(*user, *password, *url)

  session.doLogon()
  session.getManaged()
}
