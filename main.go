package main

import "fmt"

func twoSum(nums []int, target int) []int {
	ans := make([]int, 2)

	for i := 0; i < len(nums)-1; i++ {
		for j := i + 1; j < len(nums); j++ {
			if nums[i]+nums[j] == target {
				ans[0] = i
				ans[1] = j

				return ans
			}
		}
	}
	return ans
}

func main() {
	sli := []int{3, 2, 4}
	target := 6
	fmt.Println(twoSum(sli, target))
}
