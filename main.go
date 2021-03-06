package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/blizztrack/publish/core"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	awsConfig *aws.Config
	s3Client  *s3.S3

	AccessKey = kingpin.Arg("access_key", "Access key for S3").Envar("ACCESS_KEY").Required().String()
	SecretKey = kingpin.Arg("secret_key", "Secret key for S3").Envar("SECRET_KEY").Required().String()
	Endpoint  = kingpin.Arg("endpoint", "Server endpoint").Envar("ENDPOINT").Required().String()
	Region    = kingpin.Arg("region", "Region").Envar("REGION").Default("us-east-1").String()
)

func main() {
	log.Printf("CWD: %s", getCWD())

	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) > 0 {
		if argsWithoutProg[0] == "init" {
			createPubishFile()
			return
		}
	}

	kingpin.Parse()
	awsConfig = &aws.Config{
		Credentials: credentials.NewStaticCredentials(*AccessKey, *SecretKey, ""),
		Endpoint:    aws.String("https://" + *Endpoint),
		Region:      aws.String(*Region), // This is counter intuitive, but it will fail with a non-AWS region name.
	}

	newSession := session.New(awsConfig)
	s3Client = s3.New(newSession)

	readAndProcessConfig()

	log.Println("Finished publishing assets")
}

func readAndProcessConfig() {
	configFile := fmt.Sprintf("%s/%s", getCWD(), ".publish.json")
	plan, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalln(err)
		return
	}

	var conf core.ConfigFile
	json.Unmarshal(plan, &conf)

	for _, item := range conf.Files {
		if len(item.Pattern) == 0 {
			processFile(conf, item)
		} else {
			processPattern(conf, item)
		}
	}
}

func processFile(conf core.ConfigFile, item core.File) {
	acl := item.ACL
	if len(conf.GlobalACL) > 0 {
		acl = conf.GlobalACL
	}

	localPath := path.Join(getCWD(), item.Path)

	out, err := uploadToS3(conf.Bucket, localPath, item.Remote, acl)
	if err != nil {
		log.Panicln(err)
	}

	if out == nil {
		return
	}

	log.Printf("Uploaded %s to bucket %s", item.Path, conf.Bucket)
}

func processPattern(conf core.ConfigFile, item core.File) {
	acl := item.ACL
	if len(conf.GlobalACL) > 0 {
		acl = conf.GlobalACL
	}

	globPath := fmt.Sprintf("%s/%s", getCWD(), item.Pattern)
	// TODO: handle error
	matches, _ := filepath.Glob(globPath)

	for _, match := range matches {
		filePath := strings.Replace(match, getCWD(), "", -1)
		// Cause windows
		filePath = strings.Replace(filePath, "\\", "/", -1)
		item.Path = strings.Trim(item.Path, "/")

		out, _ := uploadToS3(conf.Bucket, match, item.Remote+filePath, acl)

		if out == nil {
			continue
		}
		log.Printf("Uploaded %s to bucket %s", filePath, conf.Bucket)
	}
}

// Default ACL permission to private
func uploadToS3(bucket, localpath, remotepath, permission string) (*s3.PutObjectOutput, error) {
	if strings.Contains(localpath, ".publish.json") {
		return nil, nil
	}

	if len(permission) == 0 {
		permission = "private"
	}

	file, err := os.Open(localpath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := fileInfo.Size()
	buffer := make([]byte, fileSize)
	file.Read(buffer)

	object := s3.PutObjectInput{
		Body:          bytes.NewReader(buffer),
		Bucket:        aws.String(bucket),
		Key:           aws.String(remotepath),
		ACL:           aws.String(permission),
		ContentType:   aws.String(http.DetectContentType(buffer)),
		ContentLength: aws.Int64(fileSize),
	}
	out, err := s3Client.PutObject(&object)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func createPubishFile() {
	creationPath := fmt.Sprintf("%s/%s", getCWD(), ".publish.json")

	data := core.ConfigFile{
		Bucket: "Change-Me",
		Files: []core.File{
			core.File{
				Path:   "example.txt",
				Remote: "example/example.txt",
				ACL:    "public-read",
			},
		},
	}

	publishJson, _ := json.MarshalIndent(data, "", "\t")
	ioutil.WriteFile(creationPath, publishJson, 0644)
	log.Println("Created .publish.json with example data")
}

func getCWD() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	return dir
}
