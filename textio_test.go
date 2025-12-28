package textio

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func isEqualStringSlices(ss1, ss2 []string) bool {
	if len(ss1) != len(ss2) {
		return false
	}
	for i, w1 := range ss1 {
		if w1 != ss2[i] {
			return false
		}
	}
	return true
}

func TestValidRead(t *testing.T) {
	expectedWords := []string{"hello", "world"}

	reader := NewReader()
	input := "hello \n world\n\n"
	r := strings.NewReader(input)
	reader.SetReaders(r)
	words, err := reader.ReadAll()
	if err != nil {
		t.Error(err)
	}
	if !isEqualStringSlices(words, expectedWords) {
		t.Errorf("words are different ! Expected %v, got %v\n", expectedWords, words)
	}
}

func TestInvaliddRead(t *testing.T) {
	reader := NewReader()
	reader.SetFilter(func(s string, ctx any) bool {
		alphaRegex := regexp.MustCompile(`^[A-Za-z]+$`)
		return alphaRegex.MatchString(s)
	})
	reader.FailOnInvalid = true
	input := "hellé\n"
	r := strings.NewReader(input)
	reader.SetReaders(r)
	_, err := reader.ReadAll()
	if err == nil {
		t.Error("Should have been an error because char 'é' is not valid in the default reader configuration")
	}
}

func TestFileRead(t *testing.T) {
	expectedWords := []string{"orange", "apple", "pineapple"}
	reader := NewReader()
	reader.FailOnInvalid = true
	file, err := os.Open("test.txt")
	if err != nil {
		t.Error(err)
	}
	reader.SetReaders(file)
	words, err := reader.ReadAll()
	if err != nil {
		t.Error(err)
	}
	if !isEqualStringSlices(words, expectedWords) {
		t.Errorf("words are different ! Expected %v, got %v\n", expectedWords, words)
	}
}

func TestValidInvalidRead(t *testing.T) {
	expectedWords := []string{"hello", "world"}

	reader := NewReader()
	reader.SetFilter(func(s string, ctx any) bool {
		alphaRegex := regexp.MustCompile(`^[A-Za-z]+$`)
		return alphaRegex.MatchString(s)
	})
	reader.SetNormalizer(func(s string, ctx any) string {
		return strings.TrimSpace(s)
	})
	input := "hello  \n world\n1nval1de\n\n"
	r := strings.NewReader(input)
	reader.SetReaders(r)
	words, err := reader.ReadAll()
	if err != nil {
		t.Error(err)
	}
	if !isEqualStringSlices(words, expectedWords) {
		t.Errorf("words are different ! Expected %v, got %v\n", expectedWords, words)
	}
}

func TestWeirdDelimiter(t *testing.T) {
	expectedWords := []string{"hello", "world"}

	reader := NewReader()
	reader.SetNormalizer(func(s string, ctx any) string {
		res := strings.TrimSpace(s)
		if res == " " {
			return ""
		}
		return res
	})
	reader.SetDelimiter("next")
	input := "hello next world nextnext"
	r := strings.NewReader(input)
	reader.SetReaders(r)
	words, err := reader.ReadAll()
	if err != nil {
		t.Error(err)
	}
	if !isEqualStringSlices(words, expectedWords) {
		t.Errorf("words are different ! Expected %v, got %v\n", expectedWords, words)
	}
}

func TestRead(t *testing.T) {
	p := make([]byte, 10)
	reader := NewReader()
	reader.SetReaders(strings.NewReader("hello there"))
	n, err := reader.Read(p)
	if err != nil {
		t.Error(err)
	}
	if n != 10 {
		t.Errorf("n should be equal to 10, but is %d", n)
	}
}
