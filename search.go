package spock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type jsonobject struct {
	Result ResponseType
}

type ResponseType struct {
	Status     string
	Suggestion string
	Results    []ResultType
}

type ResultType struct {
	Title     string
	Lang      string
	Highlight string
}

func (gs *GitStorage) Search(query string) (*ResponseType, error) {
	postData := make(map[string]string)
	postData["query"] = query

	jsonPostData, err := json.Marshal(postData)
	if err != nil {
		log.Printf("Error serializing JSON POST data: %s\n", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", gs.IndexServerAddr+"/search", bytes.NewReader(jsonPostData))
	if err != nil {
		log.Printf("Error creating HTTP request: %s\n", err)
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	var jsontype jsonobject
	err = json.Unmarshal(body, &jsontype)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	return &jsontype.Result, nil
}

func (gs *GitStorage) IndexDocument(title string) error {
	postData := make(map[string]string)
	postData["name"] = title
	jsonPostData, err := json.Marshal(postData)
	if err != nil {
		log.Print(err)
		return err
	}

	req, err := http.NewRequest("POST", gs.IndexServerAddr+"/add", bytes.NewReader(jsonPostData))
	if err != nil {
		log.Print(err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return err
	}

	if resp.StatusCode == 200 {
		var jsontype jsonobject
		err = json.Unmarshal(body, &jsontype)
		if err != nil {
			log.Print(err)
			return err
		}
	} else {
		return fmt.Errorf("Error in response, status code != 200\n")
	}

	return nil
}
