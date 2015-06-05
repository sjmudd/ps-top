package main

import (
	"fmt"
)

func main() {
	fmt.Println("Warning: this is not the pstop binary!")
	fmt.Println("")
	fmt.Println("As of v0.5, and due to a possible naming conflict, pstop has been")
	fmt.Println("renamed to ps-top and ps-stats is a separate binary.")
	fmt.Println("This change required me to move the build locations to a different directory.")
	fmt.Println("")
	fmt.Println("Also please adjust your github.com repo reference to https://github.com/sjmudd/ps-top.git")
	fmt.Println("")
	fmt.Println("Install each binary by doing:")
	fmt.Println("go get -u github.com/sjmudd/ps-top/cmd/ps-top")
	fmt.Println("go get -u github.com/sjmudd/ps-top/cmd/ps-stats")
	fmt.Println("")
	fmt.Println("Sorry for the confusion")
}
