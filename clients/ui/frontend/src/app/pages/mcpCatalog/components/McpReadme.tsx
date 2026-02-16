import * as React from 'react';

type McpReadmeProps = {
  content: string;
};

const escapeHtml = (text: string): string =>
  text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');

const parseMarkdown = (markdown: string): string => {
  const lines = markdown.split('\n');
  const htmlLines: string[] = [];
  let inCodeBlock = false;
  let inList = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Code block toggle
    if (line.startsWith('```')) {
      if (inCodeBlock) {
        htmlLines.push('</code></pre>');
        inCodeBlock = false;
      } else {
        if (inList) {
          htmlLines.push('</ul>');
          inList = false;
        }
        htmlLines.push('<pre><code>');
        inCodeBlock = true;
      }
      continue;
    }

    if (inCodeBlock) {
      htmlLines.push(escapeHtml(line));
      continue;
    }

    // Close list if current line is not a list item
    if (inList && !line.startsWith('- ') && !line.startsWith('* ')) {
      htmlLines.push('</ul>');
      inList = false;
    }

    // Empty line
    if (line.trim() === '') {
      htmlLines.push('');
      continue;
    }

    // Headers
    if (line.startsWith('### ')) {
      htmlLines.push(`<h3>${parseInline(escapeHtml(line.substring(4)))}</h3>`);
      continue;
    }
    if (line.startsWith('## ')) {
      htmlLines.push(`<h2>${parseInline(escapeHtml(line.substring(3)))}</h2>`);
      continue;
    }
    if (line.startsWith('# ')) {
      htmlLines.push(`<h1>${parseInline(escapeHtml(line.substring(2)))}</h1>`);
      continue;
    }

    // List items
    if (line.startsWith('- ') || line.startsWith('* ')) {
      if (!inList) {
        htmlLines.push('<ul>');
        inList = true;
      }
      const content = line.substring(2);
      htmlLines.push(`<li>${parseInline(escapeHtml(content))}</li>`);
      continue;
    }

    // Regular paragraph
    htmlLines.push(`<p>${parseInline(escapeHtml(line))}</p>`);
  }

  if (inCodeBlock) {
    htmlLines.push('</code></pre>');
  }
  if (inList) {
    htmlLines.push('</ul>');
  }

  return htmlLines.join('\n');
};

const parseInline = (text: string): string => {
  let result = text;
  // Bold: **text**
  result = result.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  // Italic: *text*
  result = result.replace(/\*(.+?)\*/g, '<em>$1</em>');
  // Inline code: `text`
  result = result.replace(/`(.+?)`/g, '<code>$1</code>');
  // Links: [text](url)
  result = result.replace(
    /\[([^\]]+)\]\(([^)]+)\)/g,
    '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>',
  );
  return result;
};

const McpReadme: React.FC<McpReadmeProps> = ({ content }) => {
  const html = React.useMemo(() => parseMarkdown(content), [content]);

  return (
    <div
      className="pf-v6-c-content"
      // eslint-disable-next-line react/no-danger
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
};

export default McpReadme;
