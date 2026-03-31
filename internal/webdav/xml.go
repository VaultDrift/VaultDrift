package webdav

import "encoding/xml"

// PropFind represents a PROPFIND request
type PropFind struct {
	XMLName  xml.Name  `xml:"propfind"`
	AllProp  *struct{} `xml:"allprop"`
	PropName *struct{} `xml:"propname"`
	Prop     *Prop     `xml:"prop"`
}

// Prop represents a list of properties
type Prop struct {
	XMLName    xml.Name      `xml:"prop"`
	Properties []PropertyDef `xml:",any"`
}

// PropertyDef represents a property definition
type PropertyDef struct {
	XMLName xml.Name
}

// PropertyUpdate represents a PROPPATCH request
type PropertyUpdate struct {
	XMLName xml.Name `xml:"propertyupdate"`
	Set     *Set     `xml:"set"`
	Remove  *Remove  `xml:"remove"`
}

// Set represents a property set operation
type Set struct {
	XMLName xml.Name `xml:"set"`
	Prop    Prop     `xml:"prop"`
}

// Remove represents a property remove operation
type Remove struct {
	XMLName xml.Name `xml:"remove"`
	Prop    Prop     `xml:"prop"`
}

// MultiStatus represents a 207 Multi-Status response
type MultiStatus struct {
	XMLName   xml.Name   `xml:"multistatus"`
	XMLNS     string     `xml:"xmlns:d,attr"`
	Responses []Response `xml:"response"`
}

// Response represents a single response in a Multi-Status
type Response struct {
	XMLName  xml.Name   `xml:"response"`
	Href     string     `xml:"href"`
	Property []Property `xml:"propstat>prop>"`
	Status   string     `xml:"propstat>status,omitempty"`
}

// Property represents a single WebDAV property
type Property struct {
	XMLName        xml.Name
	Name           string `xml:"-"`
	Value          string `xml:",chardata"`
	IsResourceType bool   `xml:"-"`
}

// LockInfo represents a lock request body
type LockInfo struct {
	XMLName   xml.Name  `xml:"lockinfo"`
	LockScope LockScope `xml:"lockscope"`
	LockType  LockType  `xml:"locktype"`
	Owner     Owner     `xml:"owner"`
	Timeout   string    `xml:"timeout,attr,omitempty"`
}

// LockScope represents the lock scope
type LockScope struct {
	Exclusive *struct{} `xml:"exclusive"`
	Shared    *struct{} `xml:"shared"`
}

// LockType represents the lock type
type LockType struct {
	Write *struct{} `xml:"write"`
}

// Owner represents the lock owner
type Owner struct {
	Href string `xml:"href"`
}

// LockDiscovery represents a lock discovery property
type LockDiscovery struct {
	XMLName    xml.Name   `xml:"lockdiscovery"`
	ActiveLock ActiveLock `xml:"activelock"`
}

// ActiveLock represents an active lock
type ActiveLock struct {
	XMLName   xml.Name  `xml:"activelock"`
	LockType  string    `xml:"locktype>write"`
	LockScope string    `xml:"lockscope>exclusive"`
	Depth     string    `xml:"depth"`
	Owner     Owner     `xml:"owner"`
	Timeout   string    `xml:"timeout"`
	LockToken LockToken `xml:"locktoken"`
	LockRoot  LockRoot  `xml:"lockroot"`
}

// LockToken represents a lock token
type LockToken struct {
	Href string `xml:"href"`
}

// LockRoot represents the lock root resource
type LockRoot struct {
	Href string `xml:"href"`
}
