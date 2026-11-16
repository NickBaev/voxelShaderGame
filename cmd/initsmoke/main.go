package main

import (
	"fmt"
	"voxelgame/compute"
)

func main() {
	ctx, err := compute.Init()
	if err != nil {
		panic(err)
	}
	fmt.Printf("instance+device+queue OK, queueFamily=%d\n", ctx.QueueFamily)
	ctx.Destroy()
	fmt.Println("destroy OK")
}
