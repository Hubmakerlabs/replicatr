package digestr

import (
	"bufio"
	"fmt"
	"os"
)

func cleanUp() {
	// Define the relative path to the file
	relativePath := "canisterInfo.txt"

	// Open the file
	file, err := os.Open(relativePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Use a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Initialize variables to hold the strings
	var cannisterAddr, cannisterId string

	// Assuming the file has at least two lines
	if scanner.Scan() {
		cannisterAddr = scanner.Text()
	}
	if scanner.Scan() {
		cannisterId = scanner.Text()
	}

	// Check for errors in scanning
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading from file: %v\n", err)
		return
	}

	// Use the strings as needed
	fmt.Printf("String 1: %s\nString 2: %s\n", cannisterAddr, cannisterId)
	// Instead of printing, you can use str1 and str2 as variables here
}
