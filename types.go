package main

// BlogMeta represents the metadata of a blog post
type BlogMeta struct {
	Date    string
	Title   string
	Author  string
	Header  string
	Summary string
	Status  string
}

// BlogPost represents a complete blog post with metadata and content
type BlogPost struct {
	Meta    BlogMeta
	Content []string
}

// BlogExtractor is the strategy interface for extracting blog posts from different formats
type BlogExtractor interface {
	Extract(doc interface{}, source []byte) (*BlogPost, bool)
}
