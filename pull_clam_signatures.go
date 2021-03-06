package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rhdedgar/clamsig-puller/config"
	"github.com/rhdedgar/clamsig-puller/models"
)

func downloadSignatures(configFile *models.ConfigFile) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(configFile.ClamBucketRegion),
		Credentials: credentials.NewStaticCredentials(configFile.ClamBucketKeyID, configFile.ClamBucketKey, ""),
	})

	svc := s3.New(sess)

	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(configFile.ClamMirrorBucket)})
	if err != nil {
		fmt.Printf("Unable to list items in bucket %q, %v", configFile.ClamMirrorBucket, err)
	}

	downloader := s3manager.NewDownloader(sess)

	// Loop through bucket contents, and compare with our json array. If file is a match, then
	// check if doesn't exist, and check if the bucket's file is newer. Download it in those cases.
	for _, item := range resp.Contents {
		for _, localItem := range configFile.ClamConfigFiles {
			if *item.Key == localItem {
				fileStat, err := os.Stat(filepath.Join(config.ClamInstallDir, localItem))
				if os.IsNotExist(err) || fileStat.ModTime().Before(*item.LastModified) {
					// Adding a little jitter, so that there's not such a thundering herd
					// of network traffic upon updating.
					rand.Seed(time.Now().UnixNano())
					n := rand.Intn(5)
					time.Sleep(time.Duration(n) * time.Second)

					newFile, err := os.Create(filepath.Join(config.ClamInstallDir, localItem))
					if err != nil {
						fmt.Printf("Unable to open file %q\n, %v\n", item, err)
					}

					defer newFile.Close()

					_, err = downloader.Download(newFile,
						&s3.GetObjectInput{
							Bucket: aws.String(configFile.ClamMirrorBucket),
							Key:    aws.String(*item.Key),
						})
					if err != nil {
						fmt.Printf("Unable to download item %q\n, %v\n", item, err)
					} else {
						fmt.Println("Downloaded the following:")
						fmt.Println("Name:         ", *item.Key)
						fmt.Println("Last modified:", *item.LastModified)
						fmt.Println("Size:         ", *item.Size, "bytes")
						fmt.Println("")
					}
				} else if err != nil {
					fmt.Println("Hit an issue opening the file:", err)
				}
			}
		}
	}
}

func main() {
	fmt.Println("ClamAV signature and config updater v0.0.1")

	newConfig := &config.ConfigFile
	downloadSignatures(newConfig)
}
