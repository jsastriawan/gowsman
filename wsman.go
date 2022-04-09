package gowsman

import (
	"fmt"
	"io"
	"reflect"
	"strconv"

	"github.com/beevik/etree"
)

var ns_prefix = map[string]string{
	"AMT": "http://intel.com/wbem/wscim/1/amt-schema/1/",
	"CIM": "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/",
	"IPS": "http://intel.com/wbem/wscim/1/ips-schema/1/",
}

var wsmanGetTemplate = `
<?xml version="1.0" encoding="UTF-8" ?>
<a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope"
	xmlns:b="http://schemas.xmlsoap.org/ws/2004/08/addressing" 
	xmlns:c="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd"> 
	<a:Header>
		<b:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/transfer/Get</b:Action>
		<b:To>/wsman</b:To>
		<c:ResourceURI>%s</c:ResourceURI>
		<b:MessageID>%s</b:MessageID>
		<b:ReplyTo><b:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</b:Address></b:ReplyTo>
		<c:OperationTimeout>PT60S</c:OperationTimeout>
	</a:Header>
	<a:Body/>
</a:Envelope>
`
var wsmanEnumTemplate = `
<?xml version="1.0" encoding="UTF-8" ?>
<a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope"
	xmlns:b="http://schemas.xmlsoap.org/ws/2004/08/addressing" 
	xmlns:c="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd"> 
	<a:Header>
		<b:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/enumeration/Enumerate</b:Action>
		<b:To>/wsman</b:To>
		<c:ResourceURI>%s</c:ResourceURI>
		<b:MessageID>%s</b:MessageID>
		<b:ReplyTo><b:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</b:Address></b:ReplyTo>
		<c:OperationTimeout>PT60S</c:OperationTimeout>
	</a:Header>
	<a:Body>
		<Enumerate xmlns="http://schemas.xmlsoap.org/ws/2004/09/enumeration" />
	</a:Body>
</a:Envelope>
`

var wsmanPullTemplate = `
<?xml version="1.0" encoding="UTF-8" ?>
<a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope"
	xmlns:b="http://schemas.xmlsoap.org/ws/2004/08/addressing" 
	xmlns:c="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd"> 
	<a:Header>
		<b:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/enumeration/Pull</b:Action>
		<b:To>/wsman</b:To>
		<c:ResourceURI>%s</c:ResourceURI>
		<b:MessageID>%s</b:MessageID>
		<b:ReplyTo><b:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</b:Address></b:ReplyTo>
		<c:OperationTimeout>PT60S</c:OperationTimeout>
	</a:Header>
	<a:Body>
		<Pull xmlns="http://schemas.xmlsoap.org/ws/2004/09/enumeration">
			<EnumerationContext>%s</EnumerationContext>
			<MaxElements>999</MaxElements>
		</Pull>
	</a:Body>
</a:Envelope>
`

var wsmanPutTemplate = `
<?xml version="1.0" encoding="UTF-8" ?>
<a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope"
	xmlns:b="http://schemas.xmlsoap.org/ws/2004/08/addressing" 
	xmlns:c="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd"> 
	<a:Header>
		<b:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/transfer/Put</b:Action>
		<b:To>/wsman</b:To>
		<c:ResourceURI>%s</c:ResourceURI>
		<b:MessageID>%s</b:MessageID>
		<b:ReplyTo><b:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</b:Address></b:ReplyTo>
		<c:OperationTimeout>PT60S</c:OperationTimeout>
		%s
	</a:Header>
	<a:Body>
%s
	</a:Body>
</a:Envelope>
`

type WSMan struct {
	Header map[string]interface{}
	Body   map[string]interface{}
}

func newWSMan() *WSMan {
	w := WSMan{}
	return &w
}

func parseChildElements(el *etree.Element) map[string]interface{} {
	// proper node structure
	var node = map[string]interface{}{}
	// track every child element as array to detect potential array
	var node_array = make(map[string][]interface{})

	for _, v := range el.ChildElements() {
		if len(v.ChildElements()) > 0 {
			var node1 = parseChildElements(v)
			if node_array[v.Tag] == nil {
				var chld_array = []interface{}{node1}
				node_array[v.Tag] = chld_array
			} else {
				node_array[v.Tag] = append(node_array[v.Tag], node1)
			}
		} else {
			val_int, err := strconv.ParseInt(v.Text(), 10, 64)
			if err == nil {
				var chld_array = []interface{}{val_int}
				node_array[v.Tag] = chld_array
				continue
			}
			val_bool, err := strconv.ParseBool(v.Text())
			if err == nil {
				var chld_array = []interface{}{val_bool}
				node_array[v.Tag] = chld_array
				continue
			}
			var chld_array = []interface{}{v.Text()}
			node_array[v.Tag] = chld_array
		}
	}
	for k, v := range node_array {
		if len(v) > 1 {
			node[k] = v
		} else {
			node[k] = v[0]
		}
	}
	return node
}

func createPutStringFromBody(objname string, body map[string]interface{}) string {
	if body == nil {
		return ""
	}
	pfx := objname[0:3]
	val, ok := ns_prefix[pfx]
	resuri := ""
	if ok {
		resuri = val + objname
	} else {
		return ""
	}
	var result = "\t\t<r:" + objname + " xmlns:r=\"" + resuri + "\">\n"
	obj := body[objname]
	for k, v := range obj.(map[string]interface{}) {
		if _, ok := v.(string); ok {
			result += "\t\t\t<r:" + k + ">" + v.(string) + "</r:" + k + ">\n"
		} else if _, ok = v.(int); ok {
			result += "\t\t\t<r:" + k + ">" + strconv.Itoa(v.(int)) + "</r:" + k + ">\n"
		} else if _, ok = v.(int64); ok {
			result += "\t\t\t<r:" + k + ">" + strconv.FormatInt(v.(int64), 10) + "</r:" + k + ">\n"
		} else if _, ok = v.(bool); ok {
			result += "\t\t\t<r:" + k + ">" + strconv.FormatBool(v.(bool)) + "</r:" + k + ">\n"
		} else if v == nil {
			result += "\t\t\t<r:" + k + "></r:" + k + ">\n"
		} else {
			fmt.Println(k + ": " + reflect.TypeOf(v).Name())
		}
	}
	result += "\t\t</r:" + objname + ">"
	return result
}

func (wsman *WSMan) ParseWsman(reader io.Reader) (*WSMan, error) {
	ws := newWSMan()
	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(reader); err != nil {
		return nil, err
	}
	root := doc.Root()
	hdr := root.FindElements("//Header")
	ws.Header = parseChildElements(hdr[0])
	body := root.FindElements("//Body")
	ws.Body = parseChildElements(body[0])
	return ws, nil
}

func (wsman WSMan) CreateWsmanGet(obj string, message_id string) string {
	wsman_str := ""
	pfx := obj[0:3]
	val, ok := ns_prefix[pfx]
	if ok {
		n_obj := val + obj
		wsman_str = fmt.Sprintf(wsmanGetTemplate, n_obj, message_id)
	}
	return wsman_str
}

func (wsman WSMan) CreateWsmanEnumerate(obj string, message_id string) string {
	wsman_str := ""
	pfx := obj[0:3]
	val, ok := ns_prefix[pfx]
	if ok {
		n_obj := val + obj
		wsman_str = fmt.Sprintf(wsmanEnumTemplate, n_obj, message_id)
	}
	return wsman_str
}

func (wsman WSMan) CreateWsmanPull(obj string, message_id string, enum_ctx string) string {
	wsman_str := ""
	pfx := obj[0:3]
	val, ok := ns_prefix[pfx]
	if ok {
		n_obj := val + obj
		wsman_str = fmt.Sprintf(wsmanPullTemplate, n_obj, message_id, enum_ctx)
	}
	return wsman_str
}

func (wsman WSMan) CreateWsmanPut(obj string, message_id string, selector string, body string) string {
	wsman_str := ""
	pfx := obj[0:3]
	val, ok := ns_prefix[pfx]
	if ok {
		n_obj := val + obj
		wsman_str = fmt.Sprintf(wsmanPutTemplate, n_obj, message_id, selector, body)
	}
	return wsman_str
}
