package KJUtil

import (
	"syscall"
	"fmt"
	"github.com/dsoprea/go-logging"
	"github.com/dsoprea/go-exif"
	"strconv"
	"net"
	"os"
	"io/ioutil"
)

func Fork() (uintptr, syscall.Errno) {
	var ret uintptr
	var err syscall.Errno
	ret, _, err = syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	return ret, err
}


type IfdEntry struct {
	IfdPath     string      `json:"ifd_path"`
	FqIfdPath   string      `json:"fq_ifd_path"`
	IfdIndex    int         `json:"ifd_index"`
	TagId       uint16      `json:"tag_id"`
	TagName     string      `json:"tag_name"`
	TagTypeId   uint16      `json:"tag_type_id"`
	TagTypeName string      `json:"tag_type_name"`
	UnitCount   uint32      `json:"unit_count"`
	Value       interface{} `json:"value"`
	ValueString string      `json:"value_string"`
}

func ExtractIPfromExIf(imgFile string) (net.IP) {
	f, err0 := os.OpenFile(imgFile, os.O_RDONLY, 0644)
	CheckErr(err0)
	data, err := ioutil.ReadAll(f)

	rawExif, err := exif.SearchAndExtractExif(data)

	if err != nil {
		if err == exif.ErrNoExif {
			fmt.Printf("EXIF data not found.\n")
			os.Exit(-1)
		}
		panic(err)
	}

	im := exif.NewIfdMappingWithStandard()
	ti := exif.NewTagIndex()
	entries := make([]IfdEntry, 0)

	visitor := func(fqIfdPath string, ifdIndex int, tagId uint16, tagType exif.TagType, valueContext exif.ValueContext)(err error) {
		defer func() {
			if state := recover(); state != nil {
				err = log.Wrap(state.(error))
				log.Panic(err)
			}
		}()

		ifdPath, err := im.StripPathPhraseIndices(fqIfdPath)
		log.PanicIf(err)

		it, err := ti.Get(ifdPath, tagId)
		if err != nil {
			if log.Is(err, exif.ErrTagNotFound) {
				fmt.Printf("WARNING: Unknown tag: [%s] (%04x)\n", ifdPath, tagId)
				return nil
			} else {
				log.Panic(err)
			}
		}

		valueString := ""
		var value interface{}
		if tagType.Type() == exif.TypeUndefined {
			var err error
			value, err = exif.UndefinedValue(ifdPath, tagId, valueContext, tagType.ByteOrder())
			if log.Is(err, exif.ErrUnhandledUnknownTypedTag) {
				value = nil
			} else if err != nil {
				log.Panic(err)
			} else {
				valueString = fmt.Sprintf("%v", value)
			}
		} else {
			valueString, err = tagType.ResolveAsString(valueContext, true)
			log.PanicIf(err)

			value = valueString
		}

		entry := IfdEntry{
			IfdPath:     ifdPath,
			FqIfdPath:   fqIfdPath,
			IfdIndex:    ifdIndex,
			TagId:       tagId,
			TagName:     it.Name,
			TagTypeId:   tagType.Type(),
			TagTypeName: tagType.Name(),
			UnitCount:   valueContext.UnitCount,
			Value:       value,
			ValueString: valueString,
		}

		entries = append(entries, entry)


		return nil
	}

	_, err = exif.Visit(exif.IfdStandard, im, ti, rawExif, visitor)

	imgId := ""
	for _, entry := range entries {
		fmt.Printf("IFD-PATH=[%s] ID=(0x%04x) NAME=[%s] COUNT=(%d) TYPE=[%s] VALUE=[%s]\n",
			entry.IfdPath, entry.TagId, entry.TagName, entry.UnitCount, entry.TagTypeName, entry.ValueString)
		if entry.TagName == "ImageUniqueID" {
			imgId = entry.ValueString
			break
		}
	}

	a, _ := strconv.ParseInt(imgId[len(imgId)-2:], 16,10)
	b, _ := strconv.ParseInt(imgId[len(imgId)-4:len(imgId)-2], 16,10)
	c, _ := strconv.ParseInt(imgId[len(imgId)-6:len(imgId)-4], 16,10)
	d, _ := strconv.ParseInt(imgId[len(imgId)-8:len(imgId)-6], 16,10)

	return net.IPv4(byte(a),byte(b), byte(c),byte(d))
}