# Converting Markdown to Microsoft Word (.docx)

This document provides instructions on how to convert the Software Requirements Specification (SRS) from Markdown format to Microsoft Word (.docx) format.

## Prerequisites

You'll need one of the following tools:

### Option 1: Pandoc (Command Line)

1. Install [Pandoc](https://pandoc.org/installing.html) for your operating system
2. Install a LaTeX distribution if you want to convert via PDF:
   - For Windows: [MiKTeX](https://miktex.org/download)
   - For macOS: [MacTeX](https://www.tug.org/mactex/mactex-download.html)
   - For Linux: TeX Live (`sudo apt-get install texlive-full` on Debian/Ubuntu)

### Option 2: Visual Studio Code with Extensions

1. Install [Visual Studio Code](https://code.visualstudio.com/)
2. Install the "Markdown All in One" extension
3. Install the "Markdown PDF" extension

### Option 3: Online Converters

- [Pandoc Online](https://pandoc.org/try/)
- [Markdown to Word](https://word2md.com/)
- [CloudConvert](https://cloudconvert.com/md-to-docx)

## Conversion Methods

### Using Pandoc (Command Line)

1. Open a terminal or command prompt
2. Navigate to the directory containing the family_service_requirements markdown file
3. Run the following command:

```bash
pandoc -s family_service_requirements.md -o family_service_requirements.docx
```

For better formatting with a reference style:

```bash
pandoc -s family_service_requirements.md -o family_service_requirements.docx --reference-doc=reference.docx
```

Note: You'll need to create a reference.docx file with your preferred styles.

### Using Visual Studio Code

1. Open the markdown file in VS Code
2. Use the "Markdown All in One" extension to preview the document
3. Use the "Markdown PDF" extension to export to PDF
4. Open the PDF in Word and save as .docx

### Using Online Converters

1. Visit one of the online converter websites
2. Upload or paste your markdown content
3. Download the converted .docx file

## Handling Images

Make sure all images referenced in the markdown file are accessible. Relative paths in the markdown file should work with Pandoc.

## Troubleshooting

- If images are missing in the output, check the image paths
- If formatting looks incorrect, try using a reference document with Pandoc
- For complex tables, you might need to adjust them manually in the final Word document
