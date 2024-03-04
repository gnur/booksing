package booksing

import (
	"embed"
	_ "embed"
)

//go:embed web/.output/public/_nuxt/*
var NuxtElements embed.FS

//go:embed web/.output/public/index.html
var NuxtIndexHTML []byte

//go:embed web/.output/public/book.png
var BookPNG []byte
