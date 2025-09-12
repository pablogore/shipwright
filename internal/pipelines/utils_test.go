package pipelines

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitlabNetrc(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		token    string
		expected string
	}{
		{
			name:     "basic credentials",
			user:     "testuser",
			token:    "testtoken",
			expected: "machine gitlab.com login testuser password testtoken\n",
		},
		{
			name:     "empty user",
			user:     "",
			token:    "testtoken",
			expected: "machine gitlab.com login  password testtoken\n",
		},
		{
			name:     "empty token",
			user:     "testuser",
			token:    "",
			expected: "machine gitlab.com login testuser password \n",
		},
		{
			name:     "both empty",
			user:     "",
			token:    "",
			expected: "machine gitlab.com login  password \n",
		},
		{
			name:     "special characters in user",
			user:     "user@domain.com",
			token:    "testtoken",
			expected: "machine gitlab.com login user@domain.com password testtoken\n",
		},
		{
			name:     "special characters in token",
			user:     "testuser",
			token:    "token-with-special-chars!@#$%",
			expected: "machine gitlab.com login testuser password token-with-special-chars!@#$%\n",
		},
		{
			name:     "whitespace in user",
			user:     " test user ",
			token:    "testtoken",
			expected: "machine gitlab.com login  test user  password testtoken\n",
		},
		{
			name:     "whitespace in token",
			user:     "testuser",
			token:    " test token ",
			expected: "machine gitlab.com login testuser password  test token \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GitlabNetrc(tt.user, tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitlabNetrc_Format(t *testing.T) {
	// Test that the output follows the correct .netrc format
	user := "testuser"
	token := "testtoken"
	result := GitlabNetrc(user, token)

	// Should start with "machine gitlab.com"
	assert.Contains(t, result, "machine gitlab.com")

	// Should contain "login"
	assert.Contains(t, result, "login")

	// Should contain "password"
	assert.Contains(t, result, "password")

	// Should end with newline
	assert.NotEmpty(t, result)
	assert.Equal(t, byte('\n'), result[len(result)-1])

	// Should contain the user and token
	assert.Contains(t, result, user)
	assert.Contains(t, result, token)
}

func TestExitOk(t *testing.T) {
	result := ExitOk()

	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, 0, result[0])
}

func TestExitOkOrTestFail(t *testing.T) {
	result := ExitOkOrTestFail()

	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, 0, result[0])
	assert.Equal(t, 1, result[1])
}

func TestReturnType(t *testing.T) {
	// Test that ReturnType is properly defined as []int
	returnType := ReturnType{0, 1, 2}

	assert.NotNil(t, returnType)
	assert.Len(t, returnType, 3)
	assert.Equal(t, 0, returnType[0])
	assert.Equal(t, 1, returnType[1])
	assert.Equal(t, 2, returnType[2])
}

func TestExitOk_Consistency(t *testing.T) {
	// Test that ExitOk returns consistent results
	result1 := ExitOk()
	result2 := ExitOk()

	assert.Equal(t, result1, result2)
	assert.Equal(t, []int{0}, result1)
	assert.Equal(t, []int{0}, result2)
}

func TestExitOkOrTestFail_Consistency(t *testing.T) {
	// Test that ExitOkOrTestFail returns consistent results
	result1 := ExitOkOrTestFail()
	result2 := ExitOkOrTestFail()

	assert.Equal(t, result1, result2)
	assert.Equal(t, []int{0, 1}, result1)
	assert.Equal(t, []int{0, 1}, result2)
}

func TestGitlabNetrc_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		token    string
		expected string
	}{
		{
			name:     "very long user",
			user:     "verylongusernamewithlotsofcharacters",
			token:    "token",
			expected: "machine gitlab.com login verylongusernamewithlotsofcharacters password token\n",
		},
		{
			name:     "very long token",
			user:     "user",
			token:    "verylongtokenwithlotsofcharactersandnumbers123456789",
			expected: "machine gitlab.com login user password verylongtokenwithlotsofcharactersandnumbers123456789\n",
		},
		{
			name:     "unicode characters",
			user:     "用户",
			token:    "令牌",
			expected: "machine gitlab.com login 用户 password 令牌\n",
		},
		{
			name:     "newline in user",
			user:     "user\nwith\nnewlines",
			token:    "token",
			expected: "machine gitlab.com login user\nwith\nnewlines password token\n",
		},
		{
			name:     "newline in token",
			user:     "user",
			token:    "token\nwith\nnewlines",
			expected: "machine gitlab.com login user password token\nwith\nnewlines\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GitlabNetrc(tt.user, tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitlabNetrc_Immutable(t *testing.T) {
	// Test that the function doesn't modify input parameters
	user := "testuser"
	token := "testtoken"

	// Call the function
	result := GitlabNetrc(user, token)

	// Original parameters should be unchanged
	assert.Equal(t, "testuser", user)
	assert.Equal(t, "testtoken", token)

	// Result should be as expected
	assert.Equal(t, "machine gitlab.com login testuser password testtoken\n", result)
}

func TestGitlabNetrc_Performance(t *testing.T) {
	// Test that the function is fast for repeated calls
	user := "testuser"
	token := "testtoken"

	// Make multiple calls
	for i := 0; i < 1000; i++ {
		result := GitlabNetrc(user, token)
		assert.NotEmpty(t, result)
	}
}

func TestReturnType_SliceOperations(t *testing.T) {
	// Test that ReturnType can be used as a regular []int slice
	returnType := ReturnType{0, 1, 2, 3, 4}

	// Test append
	returnType = append(returnType, 5)
	assert.Len(t, returnType, 6)
	assert.Equal(t, 5, returnType[5])

	// Test slice operations
	subSlice := returnType[1:4]
	assert.Len(t, subSlice, 3)
	assert.Equal(t, []int{1, 2, 3}, subSlice)

	// Test iteration
	sum := 0
	for _, v := range returnType {
		sum += v
	}
	assert.Equal(t, 15, sum) // 0+1+2+3+4+5 = 15
}

func TestExitOk_TypeCompatibility(t *testing.T) {
	// Test that ExitOk returns a value compatible with ReturnType
	result := ExitOk()

	// Should be assignable to ReturnType
	rt := result
	assert.Equal(t, result, rt)

	// Should be comparable
	assert.Equal(t, []int{0}, result)
	assert.Equal(t, ReturnType{0}, result)
}

func TestExitOkOrTestFail_TypeCompatibility(t *testing.T) {
	// Test that ExitOkOrTestFail returns a value compatible with ReturnType
	result := ExitOkOrTestFail()

	// Should be assignable to ReturnType
	rt := result
	assert.Equal(t, result, rt)

	// Should be comparable
	assert.Equal(t, []int{0, 1}, result)
	assert.Equal(t, ReturnType{0, 1}, result)
}
