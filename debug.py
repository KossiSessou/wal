from typing import List


def canPartitionKSubsets(nums: List[int], k: int) -> bool:
        
        total = sum(nums)
        target = total // k

        if target * k != total:
            return False
        memo = {}

        def solve(n, target, memo):
            if target == 0:
                return 1

            if n < 0:
                return 0

            if (n, target) in memo:
                return memo[(n, target)]
       
            if nums[n] > target:
                memo[(n, target)] = solve(n-1, target, memo)
                return memo[(n, target)]

            else:
                memo[(n, target)] = (solve(n-1, target - nums[n], memo) + solve(n-1, target, memo))
                return memo[(n, target)]

        return solve(len(nums)-1, target, memo) == k


print(canPartitionKSubsets([4, 3, 2, 3, 5, 2, 1], 4)) #true
print(canPartitionKSubsets([2,2,2,2,3,4,5], 4)) #false
print(canPartitionKSubsets([1, 2, 3, 4], 3)) #false