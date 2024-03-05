package main

type Alias string

type Union interface {
  int | string | bool
}

type Test1 struct {
  a bool
  b int
  c string
  d [2]int
  e []string
  f map[int]string
}

type Test2 struct {
  a Alias
  b Test1
}

type ignored1 interface{}

type ignored2 chan string
