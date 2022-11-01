package main

import (
	"bytes"
	"os"

	"github.com/microbus-io/fabric/errors"
)

func prepareServiceYAML() (found bool, err error) {
	fs, err := os.Stat("service.yaml")
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, errors.Trace(err)
	}

	if fs.Size() == 0 {
		_, err = createServiceYAML()
		if err != nil {
			return false, errors.Trace(err)
		}
		return false, nil // Avoids error processing an empty file
	}

	err = updateServiceYAML()
	if err != nil {
		return false, errors.Trace(err)
	}
	return true, nil
}

// createServiceYAML generates a new service.yaml file.
func createServiceYAML() (found bool, err error) {
	tt, err := LoadTemplate("service.yaml")
	if err != nil {
		return false, errors.Trace(err)
	}
	err = tt.Overwrite("service.yaml", nil)
	if err != nil {
		return false, errors.Trace(err)
	}
	printer.Printf("Service.yaml created")
	return true, nil
}

// updateServiceYAML updates the comments and sections to the latest version.
func updateServiceYAML() error {
	// Read the latest version line by line
	tt, err := LoadTemplate("service.yaml")
	if err != nil {
		return errors.Trace(err)
	}
	latest, err := tt.Execute(nil)
	if err != nil {
		return errors.Trace(err)
	}

	var commentBlock bytes.Buffer
	sectionComments := map[string]string{}
	sections := []string{}
	lines := bytes.Split(latest, []byte("\n"))
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("#")) {
			commentBlock.Write(line)
			commentBlock.WriteString("\n")
		} else {
			if commentBlock.Len() > 0 && bytes.HasSuffix(line, []byte(":")) {
				// Map the comment block to the line that follows it
				sectionComments[string(line)] = commentBlock.String()
				sections = append(sections, string(line))
			}
			commentBlock.Reset()
		}
	}

	// Read the current version line by line
	current, err := os.ReadFile("service.yaml")
	if err != nil {
		return errors.Trace(err)
	}

	var content bytes.Buffer
	commentBlock.Reset()
	lines = bytes.Split(current, []byte("\n"))
	for i, line := range lines {
		if bytes.HasPrefix(line, []byte("#")) {
			commentBlock.Write(line)
			commentBlock.WriteString("\n")
		} else {
			if bytes.HasSuffix(line, []byte(":")) && sectionComments[string(line)] != "" {
				// Print the latest comment instead of the current one
				content.WriteString(sectionComments[string(line)])
				delete(sectionComments, string(line))
			} else {
				content.Write(commentBlock.Bytes())
			}
			content.Write(line)
			if i < len(lines)-1 {
				content.WriteString("\n")
			}
			commentBlock.Reset()
		}
	}

	// Add new sections
	extraNewline := ""
	if len(lines[len(lines)-1]) > 0 {
		extraNewline = "\n"
	}
	for _, section := range sections {
		if sectionComments[string(section)] == "" {
			continue
		}
		content.WriteString(extraNewline)
		extraNewline = ""
		content.WriteString("\n")
		content.WriteString(sectionComments[string(section)])
		content.WriteString(section)
		content.WriteString("\n")
	}

	// Overwrite the original
	if !bytes.Equal(content.Bytes(), current) {
		file, err := os.Create("service.yaml")
		if err != nil {
			return err
		}
		_, err = file.Write(content.Bytes())
		file.Close()
		if err != nil {
			return err
		}
		printer.Printf("Service.yaml updated")
	}
	return nil
}
