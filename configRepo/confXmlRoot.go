package configRepo

import "encoding/xml"

type XmlRoot struct {
	XMLName xml.Name `xml:"Config"`
	IP      string   `xml:"IP"`
	Port    int      `xml:"Port"`
	NodeId  int      `xml:"NodeId"`
	Peers   []string `xml:"Peers>Peer"`
	ShowDir bool     `xml:"ShowDir"`
}

func (this *XmlRoot) FillWithXml(data []byte) error {
	err := xml.Unmarshal(data, this)
	if err != nil {
		return err
	}

	return nil
}
