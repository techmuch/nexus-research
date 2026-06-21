package main

import (
	"fmt"
	"os"

	"github.com/techmuch/nexus-research/db"
)

func main() {
	db.InitDB("nexus_test4.db")
	defer db.CloseDB()

	username := "testuser1"
	err := db.CreateUser(username, "password", true)
	if err != nil {
		fmt.Printf("user insert err: %v\n", err)
	}
	
	err = db.CreateProject(username, "proj-1", "Test Project")
	if err != nil {
		fmt.Printf("project insert err: %v\n", err)
	}

	err = db.CreateFile(username, "file-1", "proj-1", nil, "Test Map", "map")
	if err != nil {
		fmt.Printf("CreateFile error: %v\n", err)
		os.Exit(1)
	}

	err = db.UpdateFileContent(username, "file-1", `{"nodes": [], "edges": []}`)
	if err != nil {
		fmt.Printf("UpdateFileContent error: %v\n", err)
		os.Exit(1)
	}

	tree, err := db.GetFilesTree(username)
	if err != nil {
		fmt.Printf("GetFilesTree error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Success! Tree size: %d, First file content: %s\n", len(tree), tree[0].Content)
}
