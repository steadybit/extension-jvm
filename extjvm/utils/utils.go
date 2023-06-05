package utils

import "strings"

func Contains(s []int32, str int32) bool {
  for _, v := range s {
    if v == str {
      return true
    }
  }
  return false
}

func ContainsString(s []string, str string) bool {
  for _, v := range s {
    if v == str {
      return true
    }
  }
  return false
}

func ContainsPartOfString(s []string, str string) bool {
  for _, v := range s {
    if strings.Contains(v, str) {
      return true
    }
  }
  return false
}
