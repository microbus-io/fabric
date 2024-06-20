// Code generated by Microbus. DO NOT EDIT.

package resources

import "embed"

//go:embed *
var FS embed.FS

/*
Files placed in the resources directory are bundled with the executable and are accessible via svc.ResFS or
any of the convenience methods svc.ReadResFile, svc.ReadResTextFile, svc.ExecuteResTemplate, svc.ServeResFile, etc.

A file named strings.yaml can be used to store internationalized strings that can be loaded via svc.LoadResString
to best match the locale in the context. The YAML is expected to be in the following format:

stringKey:
  default: Localized
  en: Localized
  en-GB: Localised
  fr: Localisée

If a default is not provided, English (en) is used as the fallback language.
String keys and locale names are case-sensitive.
*/