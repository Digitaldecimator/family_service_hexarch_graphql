# Converting Markdown to Microsoft Word (.docx) and PDF

This document provides instructions on how to convert the Software Design Specification (SDS) from Markdown format to Microsoft Word (.docx) and PDF formats.

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
2. Navigate to the directory containing the family_service_design.md file
3. Run the following command to generate a Word document:

```bash
pandoc -s family_service_design.md -o family_service_design.docx
```

4. Run the following command to generate a PDF:

```bash
pandoc -s family_service_design.md -o family_service_design.pdf
```

For better formatting with a reference style:

```bash
pandoc -s family_service_design.md -o family_service_design.docx --reference-doc=reference.docx
```

Note: You'll need to create a reference.docx file with your preferred styles.

### Using Visual Studio Code

1. Open the markdown file in VS Code
2. Use the "Markdown All in One" extension to preview the document
3. Use the "Markdown PDF" extension to export to PDF
4. For Word format, you can either:
   - Export to PDF first, then open in Word and save as .docx
   - Use the "Markdown All in One" extension's export feature

### Using Online Converters

1. Visit one of the online converter websites
2. Upload or paste your markdown content
3. Download the converted .docx or PDF file

## Handling Images

Make sure all images referenced in the markdown file are accessible. Relative paths in the markdown file should work with Pandoc. The SDS document references diagrams in the assets/diagrams directory, so ensure these are available when converting.

## Troubleshooting

- If images are missing in the output, check the image paths
- If formatting looks incorrect, try using a reference document with Pandoc
- For complex tables, you might need to adjust them manually in the final document
- If diagrams don't render properly, consider exporting them separately and inserting them into the final document