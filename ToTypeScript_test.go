package main

import (
  "log"
  "os"
)

func ExampleGoToTypeScript() {
  var err error
  var goFile []byte
  goFile, err = os.ReadFile("./ToTypeScriptTypes_test.go")
  if err != nil {
    log.Fatal(err)
  }
  err = ConvertGoFile(goFile, os.Stdout)
  if err != nil {
    log.Fatal(err)
  }
  // Output:
  // type Alias = string
  // type Test1 = {
  //   a bool
  //   b number
  //   c string
  //   d []number
  //   e []string
  //   f Map<string, string>
  // }
  // type Test2 = {
  //   a Alias
  //   b Test1
  // }
}
