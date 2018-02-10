// Copyright 2016 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// [START storage_quickstart]
// Sample storage-quickstart creates a Google Cloud Storage bucket.
package storage

import (
	"fmt"
	"log"

	// Imports the Google Cloud Storage client package.
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	//storage "google.golang.org/api/storage/v1"
	"io/ioutil"

	"os"
)

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

type cloudStorage struct {}

func (c *cloudStorage) newStorageClient() (*storage.Client, error ) {
	ctx := context.Background()



	// Creates a client.
	client, err := storage.NewClient(ctx)

	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		return nil, err
	}
	return client, nil
}

func init() {
	// Sets your Google Cloud Platform project ID.
	//projectID := "myapisample-192900"

}

func (c *cloudStorage) Download(client *storage.Client, bucketname, filename string) error {
	ctx := context.Background()
	bh := client.Bucket(bucketname)
	// Next check if the bucket exists
	if _, err := bh.Attrs(ctx); err != nil {
		return err
	}
	obj := bh.Object(filename)
	rc, err := obj.NewReader(ctx)
	if err != nil {
		err = fmt.Errorf("readFile: unable to open file from bucket %q, file %q: %v", bucketname, filename, err)
		return err
	}

	defer rc.Close()
	slurp, err := ioutil.ReadAll(rc)
	if err != nil {
		fmt.Errorf("readFile: unable to read data from bucket %q, file %q: %v", bucketname, filename, err)
		return err
	}
	outfile := filename
	check(err)
	err2 := ioutil.WriteFile(outfile, slurp, 0666) //写入文件(字节数组)
	check(err2)

	return nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func listObject(client *storage.Client, buckets []string) (map[string][]string, error) {
	ctx := context.Background()
	var objs []string
	mobjs := make(map[string][]string, 10)

	for _, v := range buckets{
		it := client.Bucket(v).Objects(ctx, nil)
		objs = []string{}
		for {
			battrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}

			objs = append(objs, battrs.Name)

		}
		mobjs[v]= objs
	}
	return mobjs, nil
}
// [END storage_quickstart]
func list(client *storage.Client, projectID string) ([]string, error) {
	ctx := context.Background()
	// [START list_buckets]
	var buckets []string
	it := client.Buckets(ctx, projectID)
	for {
		battrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		buckets = append(buckets, battrs.Name)

	}
	// [END list_buckets]
	return buckets, nil
}