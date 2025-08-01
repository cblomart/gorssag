package poller

import (
	"strings"
	"testing"
)

// TestHTMLToMarkdown_Headings tests heading tags (h1-h6) as documented by W3C
func TestHTMLToMarkdown_Headings(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "H1 tag",
			html:     "<h1>Main Title</h1>",
			expected: "# Main Title",
		},
		{
			name:     "H2 tag",
			html:     "<h2>Subtitle</h2>",
			expected: "## Subtitle",
		},
		{
			name:     "H3 tag",
			html:     "<h3>Section</h3>",
			expected: "### Section",
		},
		{
			name:     "H4 tag",
			html:     "<h4>Subsection</h4>",
			expected: "#### Subsection",
		},
		{
			name:     "H5 tag",
			html:     "<h5>Minor section</h5>",
			expected: "##### Minor section",
		},
		{
			name:     "H6 tag",
			html:     "<h6>Smallest heading</h6>",
			expected: "###### Smallest heading",
		},
		{
			name:     "Heading with attributes",
			html:     "<h1 class='title' id='main'>Title with attributes</h1>",
			expected: "# Title with attributes",
		},
		{
			name:     "Multiple headings",
			html:     "<h1>Title</h1><h2>Subtitle</h2><p>Content</p>",
			expected: "# Title\n\n## Subtitle\n\nContent",
		},
		{
			name:     "Heading with nested content",
			html:     "<h1>Title with <strong>bold</strong> text</h1>",
			expected: "# Title with **bold** text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_Paragraphs tests paragraph tags as documented by W3C
func TestHTMLToMarkdown_Paragraphs(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Simple paragraph",
			html:     "<p>This is a paragraph.</p>",
			expected: "This is a paragraph.",
		},
		{
			name:     "Empty paragraph",
			html:     "<p></p>",
			expected: "",
		},
		{
			name:     "Paragraph with attributes",
			html:     "<p class='test'>This is a paragraph.</p>",
			expected: "This is a paragraph.",
		},
		{
			name:     "Multiple paragraphs",
			html:     "<p>First paragraph.</p><p>Second paragraph.</p>",
			expected: "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:     "Paragraph with nested elements",
			html:     "<p>Text with <strong>bold</strong> and <em>italic</em> content.</p>",
			expected: "Text with **bold** and _italic_ content.",
		},
		{
			name:     "Paragraph with only whitespace",
			html:     "<p>   </p>",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_TextFormatting tests text formatting tags as documented by W3C
func TestHTMLToMarkdown_TextFormatting(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Strong tag",
			html:     "<strong>Bold text</strong>",
			expected: "**Bold text**",
		},
		{
			name:     "B tag",
			html:     "<b>Bold text</b>",
			expected: "**Bold text**",
		},
		{
			name:     "Em tag",
			html:     "<em>Italic text</em>",
			expected: "_Italic text_",
		},
		{
			name:     "I tag",
			html:     "<i>Italic text</i>",
			expected: "_Italic text_",
		},
		{
			name:     "Nested formatting",
			html:     "<strong>Bold with <em>italic</em> inside</strong>",
			expected: "**Bold with _italic_ inside**",
		},
		{
			name:     "Multiple formatting",
			html:     "<strong>Bold</strong> and <em>italic</em> text",
			expected: "**Bold** and _italic_ text",
		},
		{
			name:     "Formatting with attributes",
			html:     "<strong class='important'>Important text</strong>",
			expected: "**Important text**",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_Links tests anchor tags as documented by W3C
func TestHTMLToMarkdown_Links(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Simple link",
			html:     "<a href='http://example.com'>Example Link</a>",
			expected: "[Example Link](http://example.com)",
		},
		{
			name:     "Link with attributes",
			html:     "<a href='http://example.com' target='_blank' rel='noopener'>Link</a>",
			expected: "[Link](http://example.com)",
		},
		{
			name:     "Link with URL as text",
			html:     "<a href='http://example.com'>http://example.com</a>",
			expected: "[http://example.com](http://example.com)",
		},
		{
			name:     "Link with domain as text",
			html:     "<a href='https://github.com/user/repo'>https://github.com/user/repo</a>",
			expected: "[https://github.com/user/repo](https://github.com/user/repo)",
		},
		{
			name:     "Link with empty text",
			html:     "<a href='http://example.com'></a>",
			expected: "",
		},
		{
			name:     "Link with whitespace text",
			html:     "<a href='http://example.com'>   </a>",
			expected: "",
		},
		{
			name:     "Link with nested formatting",
			html:     "<a href='http://example.com'>Link with <strong>bold</strong> text</a>",
			expected: "[Link with **bold** text](http://example.com)",
		},
		{
			name:     "Link without href",
			html:     "<a>Invalid link</a>",
			expected: "Invalid link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_Lists tests list tags as documented by W3C
func TestHTMLToMarkdown_Lists(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Unordered list",
			html:     "<ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul>",
			expected: "\n- Item 1\n- Item 2\n- Item 3\n\n",
		},
		{
			name:     "Ordered list",
			html:     "<ol><li>First item</li><li>Second item</li><li>Third item</li></ol>",
			expected: "1. First item\n2. Second item\n3. Third item",
		},
		{
			name:     "List with attributes",
			html:     "<ul class='menu'><li class='item'>Menu item</li></ul>",
			expected: "\n- Menu item\n\n",
		},
		{
			name:     "List with nested content",
			html:     "<ul><li>Item with <strong>bold</strong> text</li><li>Item with <a href='http://example.com'>link</a></li></ul>",
			expected: "\n- Item with **bold** text\n- Item with [link](http://example.com)\n\n",
		},
		{
			name:     "Empty list",
			html:     "<ul></ul>",
			expected: "\n\n",
		},
		{
			name:     "List with empty items",
			html:     "<ul><li></li><li>Valid item</li><li></li></ul>",
			expected: "\n- Valid item\n\n",
		},
		{
			name:     "Nested lists",
			html:     "<ul><li>Main item<ul><li>Sub item 1</li><li>Sub item 2</li></ul></li></ul>",
			expected: "- Main item\n  - Sub item 1\n  - Sub item 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_Blockquotes tests blockquote tags as documented by W3C
func TestHTMLToMarkdown_Blockquotes(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Simple blockquote",
			html:     "<blockquote>This is a quote.</blockquote>",
			expected: "\n> This is a quote.\n\n",
		},
		{
			name:     "Blockquote with attributes",
			html:     "<blockquote cite='http://example.com'>Cited quote.</blockquote>",
			expected: "\n> Cited quote.\n\n",
		},
		{
			name:     "Blockquote with nested content",
			html:     "<blockquote>Quote with <strong>bold</strong> and <em>italic</em> text.</blockquote>",
			expected: "> Quote with **bold** and _italic_ text.",
		},
		{
			name:     "Empty blockquote",
			html:     "<blockquote></blockquote>",
			expected: "",
		},
		{
			name:     "Blockquote with paragraph",
			html:     "<blockquote><p>Quoted paragraph.</p></blockquote>",
			expected: "\n> Quoted paragraph.\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_Code tests code-related tags as documented by W3C
func TestHTMLToMarkdown_Code(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Inline code",
			html:     "<code>console.log('hello')</code>",
			expected: "`console.log('hello')`",
		},
		{
			name:     "Pre block",
			html:     "<pre>function hello() {\n  console.log('world');\n}</pre>",
			expected: "\n```\nfunction hello() {\n  console.log('world');\n}\n```\n\n",
		},
		{
			name:     "Pre with code",
			html:     "<pre><code>const x = 42;</code></pre>",
			expected: "\n```\nconst x = 42;\n```\n\n",
		},
		{
			name:     "Code with attributes",
			html:     "<code class='language-javascript'>let x = 1;</code>",
			expected: "`let x = 1;`",
		},
		{
			name:     "Empty code",
			html:     "<code></code>",
			expected: "``",
		},
		{
			name:     "Code with special characters",
			html:     "<code>&lt;div&gt;Hello&lt;/div&gt;</code>",
			expected: "`<div>Hello</div>`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_Images tests image tags as documented by W3C
func TestHTMLToMarkdown_Images(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Image with meaningful alt text",
			html:     "<img src='image.jpg' alt='A beautiful sunset' />",
			expected: "**A beautiful sunset**",
		},
		{
			name:     "Image with generic alt text",
			html:     "<img src='image.jpg' alt='image' />",
			expected: "",
		},
		{
			name:     "Image with empty alt text",
			html:     "<img src='image.jpg' alt='' />",
			expected: "",
		},
		{
			name:     "Image without alt text",
			html:     "<img src='image.jpg' />",
			expected: "",
		},
		{
			name:     "Image with long alt text",
			html:     "<img src='image.jpg' alt='This is a very long alt text that exceeds the maximum allowed length for meaningful alt text in our conversion algorithm' />",
			expected: "",
		},
		{
			name:     "Image with attributes",
			html:     "<img src='image.jpg' alt='Logo' class='logo' width='100' height='50' />",
			expected: "![Logo](image.jpg)",
		},
		{
			name:     "Image with different attribute order",
			html:     "<img alt='Screenshot' src='screenshot.png' />",
			expected: "**Screenshot**",
		},
		{
			name:     "Image link",
			html:     "<a href='http://example.com'><img src='image.jpg' alt='Clickable image' /></a>",
			expected: "**Clickable image**",
		},
		{
			name:     "Image with generic terms in alt",
			html:     "<img src='image.jpg' alt='Company logo' />",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_LineBreaks tests line break tags as documented by W3C
func TestHTMLToMarkdown_LineBreaks(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "BR tag",
			html:     "Line 1<br>Line 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "BR with slash",
			html:     "Line 1<br/>Line 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "Multiple BR tags",
			html:     "Line 1<br><br>Line 3",
			expected: "Line 1\n\nLine 3",
		},
		{
			name:     "BR with attributes",
			html:     "Line 1<br class='clear'>Line 2",
			expected: "Line 1\n\nLine 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_HTMLEntities tests HTML entity decoding
func TestHTMLToMarkdown_HTMLEntities(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Ampersand",
			html:     "A &amp; B",
			expected: "A & B",
		},
		{
			name:     "Less than and greater than",
			html:     "&lt;div&gt;Hello&lt;/div&gt;",
			expected: "<div>Hello</div>",
		},
		{
			name:     "Quotes",
			html:     "&quot;Hello&quot; and &apos;World&apos;",
			expected: "\"Hello\" and 'World'",
		},
		{
			name:     "Non-breaking space",
			html:     "Hello&nbsp;World",
			expected: "Hello\u00a0World",
		},
		{
			name:     "Em dash and en dash",
			html:     "Range: 1&ndash;10, Quote: &mdash;Hello&mdash;",
			expected: "Range: 1–10, Quote: —Hello—",
		},
		{
			name:     "Ellipsis",
			html:     "And so on&hellip;",
			expected: "And so on…",
		},
		{
			name:     "Smart quotes",
			html:     "&ldquo;Left quote&rdquo; and &rsquo;Right quote&rsquo;",
			expected: "\u201cLeft quote\u201d and \u2019Right quote\u2019",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_CDATA tests CDATA sections
func TestHTMLToMarkdown_CDATA(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Simple CDATA",
			html:     "<![CDATA[<p>This is CDATA content</p>]]>",
			expected: "This is CDATA content\n\n\\]\\]>",
		},
		{
			name:     "CDATA with special characters",
			html:     "<![CDATA[<script>alert('test')</script>]]>",
			expected: "alert('test')\\]\\]>",
		},
		{
			name:     "CDATA with HTML entities",
			html:     "<![CDATA[&lt;div&gt;Content&lt;/div&gt;]]>",
			expected: "",
		},
		{
			name:     "CDATA mixed with regular content",
			html:     "Before <![CDATA[CDATA content]]> After",
			expected: "Before  After",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_ComplexDocuments tests complex HTML documents
func TestHTMLToMarkdown_ComplexDocuments(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name: "Complex article",
			html: `<article>
				<h1>Article Title</h1>
				<p>This is the <strong>introduction</strong> paragraph with a <a href="http://example.com">link</a>.</p>
				<h2>Section 1</h2>
				<p>Here's a list:</p>
				<ul>
					<li>First item</li>
					<li>Second item with <em>emphasis</em></li>
					<li>Third item</li>
				</ul>
				<blockquote>This is a quote from someone important.</blockquote>
				<p>And some <code>inline code</code> for good measure.</p>
			</article>`,
			expected: "# Article Title\n\nThis is the **introduction** paragraph with a [link](http://example.com).\n\n## Section 1\n\nHere's a list:\n\n- First item\n- Second item with _emphasis_\n- Third item\n\n> This is a quote from someone important.\n\nAnd some `inline code` for good measure.",
		},
		{
			name: "Blog post with images",
			html: `<article>
				<h1>My Blog Post</h1>
				<p>Welcome to my blog! Here's a <img src="vacation.jpg" alt="My vacation photo"> from my trip.</p>
				<p>And here's some <strong>bold text</strong> and <em>italic text</em>.</p>
				<blockquote>Life is what happens while you're busy making other plans.</blockquote>
			</article>`,
			expected: "# My Blog Post\n\nWelcome to my blog! Here's a **My vacation photo** from my trip.\n\nAnd here's some **bold text** and _italic text_.\n\n> Life is what happens while you're busy making other plans.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			// Normalize whitespace for comparison
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)
			if result != expected {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, expected)
			}
		})
	}
}

// TestHTMLToMarkdown_EdgeCases tests edge cases and error conditions
func TestHTMLToMarkdown_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Empty input",
			html:     "",
			expected: "",
		},
		{
			name:     "Only whitespace",
			html:     "   \n\t  ",
			expected: "",
		},
		{
			name:     "Plain text without HTML",
			html:     "This is plain text without any HTML tags.",
			expected: "This is plain text without any HTML tags.",
		},
		{
			name:     "Unclosed tags",
			html:     "Unclosed paragraph <strong>Bold text</strong>",
			expected: "Unclosed paragraph **Bold text**",
		},
		{
			name:     "Nested unclosed tags",
			html:     "Content <strong>Bold</strong>",
			expected: "Content **Bold**",
		},
		{
			name:     "Malformed HTML",
			html:     "<p>Content</div><span>More content",
			expected: "ContentMore content",
		},
		{
			name:     "Script tags (should be removed)",
			html:     "<script>alert('test')</script><p>Content</p>",
			expected: "Content",
		},
		{
			name:     "Style tags (should be removed)",
			html:     "<style>.class { color: red; }</style><p>Content</p>",
			expected: "Content",
		},
		{
			name:     "Comments (should be removed)",
			html:     "<!-- This is a comment --><p>Content</p>",
			expected: "Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHTMLToMarkdown(tt.html)
			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("convertHTMLToMarkdown() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestHTMLToMarkdown_Performance tests performance with large inputs
func TestHTMLToMarkdown_Performance(t *testing.T) {
	// Create a large HTML document
	largeHTML := "<div>"
	for i := 0; i < 1000; i++ {
		largeHTML += "<p>Paragraph " + string(rune(i%26+65)) + " with some <strong>bold</strong> and <em>italic</em> text.</p>"
	}
	largeHTML += "</div>"

	// This should complete quickly without hanging
	result := convertHTMLToMarkdown(largeHTML)
	if result == "" {
		t.Error("Expected non-empty result for large HTML input")
	}
}

// TestHTMLToMarkdown_Consistency tests that the same input always produces the same output
func TestHTMLToMarkdown_Consistency(t *testing.T) {
	html := "<h1>Title</h1><p>Content with <strong>bold</strong> and <em>italic</em> text.</p>"

	// Run multiple times to ensure consistency
	results := make([]string, 10)
	for i := 0; i < 10; i++ {
		results[i] = convertHTMLToMarkdown(html)
	}

	// All results should be identical
	firstResult := results[0]
	for i, result := range results {
		if result != firstResult {
			t.Errorf("Inconsistent result at iteration %d: got %q, want %q", i, result, firstResult)
		}
	}
}
