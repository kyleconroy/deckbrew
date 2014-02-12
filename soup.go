package main

import (
	"code.google.com/p/go.net/html"
	"strings"
)

func Flatten(n *html.Node) string {
	text := ""
	if n.Type == html.TextNode {
		text += n.Data
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += Flatten(c)
	}

	return text
}

func Find(n *html.Node, selector string) (*html.Node, bool) {
	return query(n, strings.Split(selector, " "))
}

func FindAll(n *html.Node, selector string) []*html.Node {
	return queryall(n, strings.Split(selector, " "))
}

func Attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func SplitTrimSpace(source, pattern string) []string {
	result := []string{}

	for _, val := range strings.Split(strings.TrimSpace(source), pattern) {
		result = append(result, strings.TrimSpace(val))
	}

	return result
}

func queryall(n *html.Node, selectors []string) []*html.Node {
	nodes := []*html.Node{}

	if len(selectors) == 0 {
		return nodes
	}

	match := false
	selector := selectors[0]

	if n.Type == html.ElementNode {
		if strings.HasPrefix(selector, ".") {
			rule := strings.Replace(selector, ".", "", 1)

			classes := strings.Split(Attr(n, "class"), " ")

			for _, class := range classes {
				if strings.TrimSpace(class) == rule {
					match = true
				}
			}
		} else if strings.HasPrefix(selector, "#") {
			match = ("#" + Attr(n, "id")) == selector
		} else {
			match = (n.Data == selector)
		}
	}

	if match {
		if len(selectors) > 1 {
			selectors = selectors[1:]
		} else {
			nodes = append(nodes, n)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		for _, node := range queryall(c, selectors) {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func query(n *html.Node, selectors []string) (*html.Node, bool) {
	if len(selectors) == 0 {
		return nil, false
	}

	match := false
	selector := selectors[0]

	if n.Type == html.ElementNode {
		// XXX An element can have multiple classes
		if strings.HasPrefix(selector, ".") {
			rule := strings.Replace(selector, ".", "", 1)

			classes := strings.Split(Attr(n, "class"), " ")

			for _, class := range classes {
				if strings.TrimSpace(class) == rule {
					match = true
				}
			}
		} else if strings.HasPrefix(selector, "#") {
			match = ("#" + Attr(n, "id")) == selector
		} else {
			match = (n.Data == selector)
		}
	}

	if match {
		if len(selectors) > 1 {
			selectors = selectors[1:]
		} else {
			return n, true
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		node, found := query(c, selectors)
		if found {
			return node, found
		}
	}

	return nil, false
}
