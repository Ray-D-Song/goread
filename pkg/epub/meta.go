package epub

import "encoding/xml"

// GetMetadata returns the metadata of the EPUB
func (e *Epub) GetMetadata() (*Metadata, error) {
	var pkg Package

	rootFile, err := e.File.Open(e.RootFile)
	if err != nil {
		return nil, err
	}
	defer rootFile.Close()

	decoder := xml.NewDecoder(rootFile)
	err = decoder.Decode(&pkg)
	if err != nil {
		return nil, err
	}

	metadata := &Metadata{}

	for _, item := range pkg.Metadata {
		tagName := item.XMLName.Local

		switch tagName {
		case "title":
			metadata.Title = item.Content
		case "creator":
			metadata.Creator = item.Content
		case "publisher":
			metadata.Publisher = item.Content
		case "language":
			metadata.Language = item.Content
		case "identifier":
			metadata.Identifier = item.Content
		case "date":
			metadata.Date = item.Content
		case "description":
			metadata.Description = item.Content
		case "rights":
			metadata.Rights = item.Content
		default:
			metadata.OtherMeta = append(metadata.OtherMeta, []string{tagName, item.Content})
		}
	}

	return metadata, nil
}
