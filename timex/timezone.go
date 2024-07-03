/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package timex

import "strings"

// TimeZoneNames is an alphabetically sorted list of most time zone names.
var TimeZoneNames = []string{
	"Africa/Abidjan",
	"Africa/Accra",
	"Africa/Addis_Ababa",
	"Africa/Algiers",
	"Africa/Asmara",
	"Africa/Asmera",
	"Africa/Bamako",
	"Africa/Bangui",
	"Africa/Banjul",
	"Africa/Bissau",
	"Africa/Blantyre",
	"Africa/Brazzaville",
	"Africa/Bujumbura",
	"Africa/Cairo",
	"Africa/Casablanca",
	"Africa/Ceuta",
	"Africa/Conakry",
	"Africa/Dakar",
	"Africa/Dar_es_Salaam",
	"Africa/Djibouti",
	"Africa/Douala",
	"Africa/El_Aaiun",
	"Africa/Freetown",
	"Africa/Gaborone",
	"Africa/Harare",
	"Africa/Johannesburg",
	"Africa/Juba",
	"Africa/Kampala",
	"Africa/Khartoum",
	"Africa/Kigali",
	"Africa/Kinshasa",
	"Africa/Lagos",
	"Africa/Libreville",
	"Africa/Lome",
	"Africa/Luanda",
	"Africa/Lubumbashi",
	"Africa/Lusaka",
	"Africa/Malabo",
	"Africa/Maputo",
	"Africa/Maseru",
	"Africa/Mbabane",
	"Africa/Mogadishu",
	"Africa/Monrovia",
	"Africa/Nairobi",
	"Africa/Ndjamena",
	"Africa/Niamey",
	"Africa/Nouakchott",
	"Africa/Ouagadougou",
	"Africa/Porto-Novo",
	"Africa/Sao_Tome",
	"Africa/Timbuktu",
	"Africa/Tripoli",
	"Africa/Tunis",
	"Africa/Windhoek",
	"America/Adak",
	"America/Anchorage",
	"America/Anguilla",
	"America/Antigua",
	"America/Araguaina",
	"America/Argentina/Buenos_Aires",
	"America/Argentina/Catamarca",
	"America/Argentina/ComodRivadavia",
	"America/Argentina/Cordoba",
	"America/Argentina/Jujuy",
	"America/Argentina/La_Rioja",
	"America/Argentina/Mendoza",
	"America/Argentina/Rio_Gallegos",
	"America/Argentina/Salta",
	"America/Argentina/San_Juan",
	"America/Argentina/San_Luis",
	"America/Argentina/Tucuman",
	"America/Argentina/Ushuaia",
	"America/Aruba",
	"America/Asuncion",
	"America/Atikokan",
	"America/Atka",
	"America/Bahia",
	"America/Bahia_Banderas",
	"America/Barbados",
	"America/Belem",
	"America/Belize",
	"America/Blanc-Sablon",
	"America/Boa_Vista",
	"America/Bogota",
	"America/Boise",
	"America/Buenos_Aires",
	"America/Cambridge_Bay",
	"America/Campo_Grande",
	"America/Cancun",
	"America/Caracas",
	"America/Catamarca",
	"America/Cayenne",
	"America/Cayman",
	"America/Chicago",
	"America/Chihuahua",
	"America/Coral_Harbour",
	"America/Cordoba",
	"America/Costa_Rica",
	"America/Creston",
	"America/Cuiaba",
	"America/Curacao",
	"America/Danmarkshavn",
	"America/Dawson",
	"America/Dawson_Creek",
	"America/Denver",
	"America/Detroit",
	"America/Dominica",
	"America/Edmonton",
	"America/Eirunepe",
	"America/El_Salvador",
	"America/Ensenada",
	"America/Fort_Nelson",
	"America/Fort_Wayne",
	"America/Fortaleza",
	"America/Glace_Bay",
	"America/Godthab",
	"America/Goose_Bay",
	"America/Grand_Turk",
	"America/Grenada",
	"America/Guadeloupe",
	"America/Guatemala",
	"America/Guayaquil",
	"America/Guyana",
	"America/Halifax",
	"America/Havana",
	"America/Hermosillo",
	"America/Indiana/Indianapolis",
	"America/Indiana/Knox",
	"America/Indiana/Marengo",
	"America/Indiana/Petersburg",
	"America/Indiana/Tell_City",
	"America/Indiana/Vevay",
	"America/Indiana/Vincennes",
	"America/Indiana/Winamac",
	"America/Indianapolis",
	"America/Inuvik",
	"America/Iqaluit",
	"America/Jamaica",
	"America/Jujuy",
	"America/Juneau",
	"America/Kentucky/Louisville",
	"America/Kentucky/Monticello",
	"America/Knox_IN",
	"America/Kralendijk",
	"America/La_Paz",
	"America/Lima",
	"America/Los_Angeles",
	"America/Louisville",
	"America/Lower_Princes",
	"America/Maceio",
	"America/Managua",
	"America/Manaus",
	"America/Marigot",
	"America/Martinique",
	"America/Matamoros",
	"America/Mazatlan",
	"America/Mendoza",
	"America/Menominee",
	"America/Merida",
	"America/Metlakatla",
	"America/Mexico_City",
	"America/Miquelon",
	"America/Moncton",
	"America/Monterrey",
	"America/Montevideo",
	"America/Montreal",
	"America/Montserrat",
	"America/Nassau",
	"America/New_York",
	"America/Nipigon",
	"America/Nome",
	"America/Noronha",
	"America/North_Dakota/Beulah",
	"America/North_Dakota/Center",
	"America/North_Dakota/New_Salem",
	"America/Nuuk",
	"America/Ojinaga",
	"America/Panama",
	"America/Pangnirtung",
	"America/Paramaribo",
	"America/Phoenix",
	"America/Port-au-Prince",
	"America/Port_of_Spain",
	"America/Porto_Acre",
	"America/Porto_Velho",
	"America/Puerto_Rico",
	"America/Punta_Arenas",
	"America/Rainy_River",
	"America/Rankin_Inlet",
	"America/Recife",
	"America/Regina",
	"America/Resolute",
	"America/Rio_Branco",
	"America/Rosario",
	"America/Santa_Isabel",
	"America/Santarem",
	"America/Santiago",
	"America/Santo_Domingo",
	"America/Sao_Paulo",
	"America/Scoresbysund",
	"America/Shiprock",
	"America/Sitka",
	"America/St_Barthelemy",
	"America/St_Johns",
	"America/St_Kitts",
	"America/St_Lucia",
	"America/St_Thomas",
	"America/St_Vincent",
	"America/Swift_Current",
	"America/Tegucigalpa",
	"America/Thule",
	"America/Thunder_Bay",
	"America/Tijuana",
	"America/Toronto",
	"America/Tortola",
	"America/Vancouver",
	"America/Virgin",
	"America/Whitehorse",
	"America/Winnipeg",
	"America/Yakutat",
	"America/Yellowknife",
	"Antarctica/Casey",
	"Antarctica/Davis",
	"Antarctica/DumontDUrville",
	"Antarctica/Macquarie",
	"Antarctica/Mawson",
	"Antarctica/McMurdo",
	"Antarctica/Palmer",
	"Antarctica/Rothera",
	"Antarctica/South_Pole",
	"Antarctica/Syowa",
	"Antarctica/Troll",
	"Antarctica/Vostok",
	"Arctic/Longyearbyen",
	"Asia/Aden",
	"Asia/Almaty",
	"Asia/Amman",
	"Asia/Anadyr",
	"Asia/Aqtau",
	"Asia/Aqtobe",
	"Asia/Ashgabat",
	"Asia/Ashkhabad",
	"Asia/Atyrau",
	"Asia/Baghdad",
	"Asia/Bahrain",
	"Asia/Baku",
	"Asia/Bangkok",
	"Asia/Barnaul",
	"Asia/Beirut",
	"Asia/Bishkek",
	"Asia/Brunei",
	"Asia/Calcutta",
	"Asia/Chita",
	"Asia/Choibalsan",
	"Asia/Chongqing",
	"Asia/Chungking",
	"Asia/Colombo",
	"Asia/Dacca",
	"Asia/Damascus",
	"Asia/Dhaka",
	"Asia/Dili",
	"Asia/Dubai",
	"Asia/Dushanbe",
	"Asia/Famagusta",
	"Asia/Gaza",
	"Asia/Harbin",
	"Asia/Hebron",
	"Asia/Ho_Chi_Minh",
	"Asia/Hong_Kong",
	"Asia/Hovd",
	"Asia/Irkutsk",
	"Asia/Istanbul",
	"Asia/Jakarta",
	"Asia/Jayapura",
	"Asia/Jerusalem",
	"Asia/Kabul",
	"Asia/Kamchatka",
	"Asia/Karachi",
	"Asia/Kashgar",
	"Asia/Kathmandu",
	"Asia/Katmandu",
	"Asia/Khandyga",
	"Asia/Kolkata",
	"Asia/Krasnoyarsk",
	"Asia/Kuala_Lumpur",
	"Asia/Kuching",
	"Asia/Kuwait",
	"Asia/Macao",
	"Asia/Macau",
	"Asia/Magadan",
	"Asia/Makassar",
	"Asia/Manila",
	"Asia/Muscat",
	"Asia/Nicosia",
	"Asia/Novokuznetsk",
	"Asia/Novosibirsk",
	"Asia/Omsk",
	"Asia/Oral",
	"Asia/Phnom_Penh",
	"Asia/Pontianak",
	"Asia/Pyongyang",
	"Asia/Qatar",
	"Asia/Qostanay",
	"Asia/Qyzylorda",
	"Asia/Rangoon",
	"Asia/Riyadh",
	"Asia/Saigon",
	"Asia/Sakhalin",
	"Asia/Samarkand",
	"Asia/Seoul",
	"Asia/Shanghai",
	"Asia/Singapore",
	"Asia/Srednekolymsk",
	"Asia/Taipei",
	"Asia/Tashkent",
	"Asia/Tbilisi",
	"Asia/Tehran",
	"Asia/Tel_Aviv",
	"Asia/Thimbu",
	"Asia/Thimphu",
	"Asia/Tokyo",
	"Asia/Tomsk",
	"Asia/Ujung_Pandang",
	"Asia/Ulaanbaatar",
	"Asia/Ulan_Bator",
	"Asia/Urumqi",
	"Asia/Ust-Nera",
	"Asia/Vientiane",
	"Asia/Vladivostok",
	"Asia/Yakutsk",
	"Asia/Yangon",
	"Asia/Yekaterinburg",
	"Asia/Yerevan",
	"Atlantic/Azores",
	"Atlantic/Bermuda",
	"Atlantic/Canary",
	"Atlantic/Cape_Verde",
	"Atlantic/Faeroe",
	"Atlantic/Faroe",
	"Atlantic/Jan_Mayen",
	"Atlantic/Madeira",
	"Atlantic/Reykjavik",
	"Atlantic/South_Georgia",
	"Atlantic/St_Helena",
	"Atlantic/Stanley",
	"Australia/ACT",
	"Australia/Adelaide",
	"Australia/Brisbane",
	"Australia/Broken_Hill",
	"Australia/Canberra",
	"Australia/Currie",
	"Australia/Darwin",
	"Australia/Eucla",
	"Australia/Hobart",
	"Australia/LHI",
	"Australia/Lindeman",
	"Australia/Lord_Howe",
	"Australia/Melbourne",
	"Australia/NSW",
	"Australia/North",
	"Australia/Perth",
	"Australia/Queensland",
	"Australia/South",
	"Australia/Sydney",
	"Australia/Tasmania",
	"Australia/Victoria",
	"Australia/West",
	"Australia/Yancowinna",
	"Brazil/Acre",
	"Brazil/DeNoronha",
	"Brazil/East",
	"Brazil/West",
	"Canada/Atlantic",
	"Canada/Central",
	"Canada/Eastern",
	"Canada/Mountain",
	"Canada/Newfoundland",
	"Canada/Pacific",
	"Canada/Saskatchewan",
	"Canada/Yukon",
	"Chile/Continental",
	"Chile/EasterIsland",
	"Europe/Amsterdam",
	"Europe/Andorra",
	"Europe/Astrakhan",
	"Europe/Athens",
	"Europe/Belfast",
	"Europe/Belgrade",
	"Europe/Berlin",
	"Europe/Bratislava",
	"Europe/Brussels",
	"Europe/Bucharest",
	"Europe/Budapest",
	"Europe/Busingen",
	"Europe/Chisinau",
	"Europe/Copenhagen",
	"Europe/Dublin",
	"Europe/Gibraltar",
	"Europe/Guernsey",
	"Europe/Helsinki",
	"Europe/Isle_of_Man",
	"Europe/Istanbul",
	"Europe/Jersey",
	"Europe/Kaliningrad",
	"Europe/Kiev",
	"Europe/Kirov",
	"Europe/Lisbon",
	"Europe/Ljubljana",
	"Europe/London",
	"Europe/Luxembourg",
	"Europe/Madrid",
	"Europe/Malta",
	"Europe/Mariehamn",
	"Europe/Minsk",
	"Europe/Monaco",
	"Europe/Moscow",
	"Europe/Nicosia",
	"Europe/Oslo",
	"Europe/Paris",
	"Europe/Podgorica",
	"Europe/Prague",
	"Europe/Riga",
	"Europe/Rome",
	"Europe/Samara",
	"Europe/San_Marino",
	"Europe/Sarajevo",
	"Europe/Saratov",
	"Europe/Simferopol",
	"Europe/Skopje",
	"Europe/Sofia",
	"Europe/Stockholm",
	"Europe/Tallinn",
	"Europe/Tirane",
	"Europe/Tiraspol",
	"Europe/Ulyanovsk",
	"Europe/Uzhgorod",
	"Europe/Vaduz",
	"Europe/Vatican",
	"Europe/Vienna",
	"Europe/Vilnius",
	"Europe/Volgograd",
	"Europe/Warsaw",
	"Europe/Zagreb",
	"Europe/Zaporozhye",
	"Europe/Zurich",
	"Indian/Antananarivo",
	"Indian/Chagos",
	"Indian/Christmas",
	"Indian/Cocos",
	"Indian/Comoro",
	"Indian/Kerguelen",
	"Indian/Mahe",
	"Indian/Maldives",
	"Indian/Mauritius",
	"Indian/Mayotte",
	"Indian/Reunion",
	"Mexico/BajaNorte",
	"Mexico/BajaSur",
	"Mexico/General",
	"Pacific/Apia",
	"Pacific/Auckland",
	"Pacific/Bougainville",
	"Pacific/Chatham",
	"Pacific/Chuuk",
	"Pacific/Easter",
	"Pacific/Efate",
	"Pacific/Enderbury",
	"Pacific/Fakaofo",
	"Pacific/Fiji",
	"Pacific/Funafuti",
	"Pacific/Galapagos",
	"Pacific/Gambier",
	"Pacific/Guadalcanal",
	"Pacific/Guam",
	"Pacific/Honolulu",
	"Pacific/Johnston",
	"Pacific/Kiritimati",
	"Pacific/Kosrae",
	"Pacific/Kwajalein",
	"Pacific/Majuro",
	"Pacific/Marquesas",
	"Pacific/Midway",
	"Pacific/Nauru",
	"Pacific/Niue",
	"Pacific/Norfolk",
	"Pacific/Noumea",
	"Pacific/Pago_Pago",
	"Pacific/Palau",
	"Pacific/Pitcairn",
	"Pacific/Pohnpei",
	"Pacific/Ponape",
	"Pacific/Port_Moresby",
	"Pacific/Rarotonga",
	"Pacific/Saipan",
	"Pacific/Samoa",
	"Pacific/Tahiti",
	"Pacific/Tarawa",
	"Pacific/Tongatapu",
	"Pacific/Truk",
	"Pacific/Wake",
	"Pacific/Wallis",
	"Pacific/Yap",
	"US/Alaska",
	"US/Aleutian",
	"US/Arizona",
	"US/Central",
	"US/East-Indiana",
	"US/Eastern",
	"US/Hawaii",
	"US/Indiana-Starke",
	"US/Michigan",
	"US/Mountain",
	"US/Pacific",
	"US/Samoa",
	"UTC",
}

// USTimeZoneNames is an alphabetically sorted list of US time zone names.
var USTimeZoneNames = []string{
	"US/Alaska",
	"US/Aleutian",
	"US/Arizona",
	"US/Central",
	"US/East-Indiana",
	"US/Eastern",
	"US/Hawaii",
	"US/Indiana-Starke",
	"US/Michigan",
	"US/Mountain",
	"US/Pacific",
	"US/Samoa",
}

var geoMap = map[string]string{
	"AF" /* Afghanistan */ :                                  "Asia/Kabul",                     // UTC +04:30
	"AL" /* Albania */ :                                      "Europe/Tirane",                  // UTC +01:00
	"DZ" /* Algeria */ :                                      "Africa/Algiers",                 // UTC +01:00
	"AS" /* American Samoa */ :                               "Pacific/Pago_Pago",              // UTC -11:00
	"AD" /* Andorra */ :                                      "Europe/Andorra",                 // UTC +01:00
	"AO" /* Angola */ :                                       "Africa/Luanda",                  // UTC +01:00
	"AI" /* Anguilla */ :                                     "America/Anguilla",               // UTC -04:00
	"AQ" /* Antarctica */ :                                   "Antarctica/Troll",               // UTC
	"AG" /* Antigua and Barbuda */ :                          "America/Antigua",                // UTC -04:00
	"AR" /* Argentina */ :                                    "America/Argentina/Buenos_Aires", // UTC -03:00
	"AM" /* Armenia */ :                                      "Asia/Yerevan",                   // UTC +04:00
	"AW" /* Aruba */ :                                        "America/Aruba",                  // UTC -04:00
	"AU" /* Australia */ :                                    "Australia/Sydney",               // UTC +11:00
	"AU-NS" /* Australia */ :                                 "Australia/Sydney",               // UTC +11:00
	"AU-NSW" /* Australia */ :                                "Australia/Sydney",               // UTC +11:00
	"AU-CT" /* Australia */ :                                 "Australia/Sydney",               // UTC +11:00
	"AU-ACT" /* Australia */ :                                "Australia/Sydney",               // UTC +11:00
	"AU-SA" /* Australia */ :                                 "Australia/Adelaide",             // UTC +10:30
	"AU-QL" /* Australia */ :                                 "Australia/Brisbane",             // UTC +10:00
	"AU-QLD" /* Australia */ :                                "Australia/Brisbane",             // UTC +10:00
	"AU-NT" /* Australia */ :                                 "Australia/Darwin",               // UTC +09:30
	"AU-TS" /* Australia */ :                                 "Australia/Hobart",               // UTC +11:00
	"AU-TAS" /* Australia */ :                                "Australia/Hobart",               // UTC +11:00
	"AU-VI" /* Australia */ :                                 "Australia/Melbourne",            // UTC +11:00
	"AU-VIC" /* Australia */ :                                "Australia/Melbourne",            // UTC +11:00
	"AU-WA" /* Australia */ :                                 "Australia/Perth",                // UTC +08:00
	"AT" /* Austria */ :                                      "Europe/Vienna",                  // UTC +01:00
	"AZ" /* Azerbaijan */ :                                   "Asia/Baku",                      // UTC +04:00
	"BS" /* Bahamas */ :                                      "America/Nassau",                 // UTC -05:00
	"BH" /* Bahrain */ :                                      "Asia/Bahrain",                   // UTC +03:00
	"BD" /* Bangladesh */ :                                   "Asia/Dhaka",                     // UTC +06:00
	"BB" /* Barbados */ :                                     "America/Barbados",               // UTC -04:00
	"BY" /* Belarus */ :                                      "Europe/Minsk",                   // UTC +03:00
	"BE" /* Belgium */ :                                      "Europe/Brussels",                // UTC +01:00
	"BZ" /* Belize */ :                                       "America/Belize",                 // UTC -06:00
	"BJ" /* Benin */ :                                        "Africa/Porto-Novo",              // UTC +01:00
	"BM" /* Bermuda */ :                                      "Atlantic/Bermuda",               // UTC -04:00
	"BT" /* Bhutan */ :                                       "Asia/Thimphu",                   // UTC +06:00
	"BO" /* Bolivia, Plurinational State of */ :              "America/La_Paz",                 // UTC -04:00
	"BQ" /* Bonaire, Sint Eustatius and Saba */ :             "America/Kralendijk",             // UTC -04:00
	"BA" /* Bosnia and Herzegovina */ :                       "Europe/Sarajevo",                // UTC +01:00
	"BW" /* Botswana */ :                                     "Africa/Gaborone",                // UTC +02:00
	"BR" /* Brazil */ :                                       "America/Sao_Paulo",              // UTC -03:00
	"BR-AC" /* Brazil */ :                                    "America/Rio_Branco",             // UTC -05:00
	"BR-AL" /* Brazil */ :                                    "America/Maceio",                 // UTC -03:00
	"BR-AP" /* Brazil */ :                                    "America/Belem",                  // UTC -03:00
	"BR-AM" /* Brazil */ :                                    "America/Manaus",                 // UTC -04:00
	"BR-BA" /* Brazil */ :                                    "America/Bahia",                  // UTC -03:00
	"BR-CE" /* Brazil */ :                                    "America/Fortaleza",              // UTC -03:00
	"BR-DF" /* Brazil */ :                                    "America/Sao_Paulo",              // UTC -03:00
	"BR-ES" /* Brazil */ :                                    "America/Sao_Paulo",              // UTC -03:00
	"BR-GO" /* Brazil */ :                                    "America/Sao_Paulo",              // UTC -03:00
	"BR-MA" /* Brazil */ :                                    "America/Fortaleza",              // UTC -03:00
	"BR-MG" /* Brazil */ :                                    "America/Sao_Paulo",              // UTC -03:00
	"BR-MT" /* Brazil */ :                                    "America/Cuiaba",                 // UTC -04:00
	"BR-MS" /* Brazil */ :                                    "America/Campo_Grande",           // UTC -04:00
	"BR-PA" /* Brazil */ :                                    "America/Belem",                  // UTC -03:00
	"BR-PB" /* Brazil */ :                                    "America/Fortaleza",              // UTC -03:00
	"BR-PE" /* Brazil */ :                                    "America/Recife",                 // UTC -03:00
	"BR-PI" /* Brazil */ :                                    "America/Fortaleza",              // UTC -03:00
	"BR-PR" /* Brazil */ :                                    "America/Sao_Paulo",              // UTC -03:00
	"BR-RJ" /* Brazil */ :                                    "America/Sao_Paulo",              // UTC -03:00
	"BR-RN" /* Brazil */ :                                    "America/Fortaleza",              // UTC -03:00
	"BR-RO" /* Brazil */ :                                    "America/Porto_Velho",            // UTC -04:00
	"BR-RR" /* Brazil */ :                                    "America/Boa_Vista",              // UTC -04:00
	"BR-RS" /* Brazil */ :                                    "America/Sao_Paulo",              // UTC -03:00
	"BR-SC" /* Brazil */ :                                    "America/Sao_Paulo",              // UTC -03:00
	"BR-SE" /* Brazil */ :                                    "America/Maceio",                 // UTC -03:00
	"BR-SP" /* Brazil */ :                                    "America/Sao_Paulo",              // UTC -03:00
	"BR-TO" /* Brazil */ :                                    "America/Araguaina",              // UTC -03:00
	"IO" /* British Indian Ocean Territory */ :               "Indian/Chagos",                  // UTC +06:00
	"BN" /* Brunei Darussalam */ :                            "Asia/Brunei",                    // UTC +08:00
	"BG" /* Bulgaria */ :                                     "Europe/Sofia",                   // UTC +02:00
	"BF" /* Burkina Faso */ :                                 "Africa/Ouagadougou",             // UTC
	"BI" /* Burundi */ :                                      "Africa/Bujumbura",               // UTC +02:00
	"KH" /* Cambodia */ :                                     "Asia/Phnom_Penh",                // UTC +07:00
	"CM" /* Cameroon */ :                                     "Africa/Douala",                  // UTC +01:00
	"CA" /* Canada */ :                                       "America/Toronto",                // UTC -05:00
	"CA-AB" /* Canada */ :                                    "America/Edmonton",               // UTC -07:00
	"CA-NT" /* Canada */ :                                    "America/Edmonton",               // UTC -07:00
	"CA-NS" /* Canada */ :                                    "America/Halifax",                // UTC -04:00
	"CA-PE" /* Canada */ :                                    "America/Halifax",                // UTC -04:00
	"CA-NU" /* Canada */ :                                    "America/Iqaluit",                // UTC -05:00
	"CA-NB" /* Canada */ :                                    "America/Moncton",                // UTC -04:00
	"CA-SK" /* Canada */ :                                    "America/Regina",                 // UTC -06:00
	"CA-NL" /* Canada */ :                                    "America/St_Johns",               // UTC -03:30
	"CA-ON" /* Canada */ :                                    "America/Toronto",                // UTC -05:00
	"CA-QC" /* Canada */ :                                    "America/Toronto",                // UTC -05:00
	"CA-BC" /* Canada */ :                                    "America/Vancouver",              // UTC -08:00
	"CA-YT" /* Canada */ :                                    "America/Whitehorse",             // UTC -07:00
	"CA-MB" /* Canada */ :                                    "America/Winnipeg",               // UTC -06:00
	"CV" /* Cape Verde */ :                                   "Atlantic/Cape_Verde",            // UTC -01:00
	"KY" /* Cayman Islands */ :                               "America/Cayman",                 // UTC -05:00
	"CF" /* Central African Republic */ :                     "Africa/Bangui",                  // UTC +01:00
	"TD" /* Chad */ :                                         "Africa/Ndjamena",                // UTC +01:00
	"CL" /* Chile */ :                                        "America/Santiago",               // UTC -03:00
	"CN" /* China */ :                                        "Asia/Shanghai",                  // UTC +08:00
	"CX" /* Christmas Island */ :                             "Indian/Christmas",               // UTC +07:00
	"CC" /* Cocos (Keeling) Islands */ :                      "Indian/Cocos",                   // UTC +06:30
	"CO" /* Colombia */ :                                     "America/Bogota",                 // UTC -05:00
	"KM" /* Comoros */ :                                      "Indian/Comoro",                  // UTC +03:00
	"CG" /* Congo */ :                                        "Africa/Brazzaville",             // UTC +01:00
	"CD" /* Congo, the Democratic Republic of the */ :        "Africa/Kinshasa",                // UTC +01:00
	"CD-KN" /* Congo, the Democratic Republic of the */ :     "Africa/Kinshasa",                // UTC +01:00
	"CD-BC" /* Congo, the Democratic Republic of the */ :     "Africa/Kinshasa",                // UTC +01:00
	"CD-KG" /* Congo, the Democratic Republic of the */ :     "Africa/Kinshasa",                // UTC +01:00
	"CD-KL" /* Congo, the Democratic Republic of the */ :     "Africa/Kinshasa",                // UTC +01:00
	"CD-MN" /* Congo, the Democratic Republic of the */ :     "Africa/Kinshasa",                // UTC +01:00
	"CD-KS" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-KC" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-KE" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-LO" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-SA" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-MA" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-SK" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-NK" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-IT" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-HU" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-TO" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-BU" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-NU" /* Congo, the Democratic Republic of the */ :     "Africa/Kinshasa",                // UTC +01:00
	"CD-MO" /* Congo, the Democratic Republic of the */ :     "Africa/Kinshasa",                // UTC +01:00
	"CD-SU" /* Congo, the Democratic Republic of the */ :     "Africa/Kinshasa",                // UTC +01:00
	"CD-EQ" /* Congo, the Democratic Republic of the */ :     "Africa/Kinshasa",                // UTC +01:00
	"CD-TU" /* Congo, the Democratic Republic of the */ :     "Africa/Kinshasa",                // UTC +01:00
	"CD-TA" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-HL" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-LU" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CD-HK" /* Congo, the Democratic Republic of the */ :     "Africa/Lubumbashi",              // UTC +02:00
	"CK" /* Cook Islands */ :                                 "Pacific/Rarotonga",              // UTC -10:00
	"CR" /* Costa Rica */ :                                   "America/Costa_Rica",             // UTC -06:00
	"HR" /* Croatia */ :                                      "Europe/Zagreb",                  // UTC +01:00
	"CU" /* Cuba */ :                                         "America/Havana",                 // UTC -05:00
	"CW" /* Curaçao */ :                                      "America/Curacao",                // UTC -04:00
	"CY" /* Cyprus */ :                                       "Asia/Nicosia",                   // UTC +02:00
	"CZ" /* Czech Republic */ :                               "Europe/Prague",                  // UTC +01:00
	"CI" /* Côte d'Ivoire */ :                                "Africa/Abidjan",                 // UTC
	"DK" /* Denmark */ :                                      "Europe/Copenhagen",              // UTC +01:00
	"DJ" /* Djibouti */ :                                     "Africa/Djibouti",                // UTC +03:00
	"DM" /* Dominica */ :                                     "America/Dominica",               // UTC -04:00
	"DO" /* Dominican Republic */ :                           "America/Santo_Domingo",          // UTC -04:00
	"EC" /* Ecuador */ :                                      "America/Guayaquil",              // UTC -05:00
	"EG" /* Egypt */ :                                        "Africa/Cairo",                   // UTC +02:00
	"SV" /* El Salvador */ :                                  "America/El_Salvador",            // UTC -06:00
	"GQ" /* Equatorial Guinea */ :                            "Africa/Malabo",                  // UTC +01:00
	"ER" /* Eritrea */ :                                      "Africa/Asmara",                  // UTC +03:00
	"EE" /* Estonia */ :                                      "Europe/Tallinn",                 // UTC +02:00
	"ET" /* Ethiopia */ :                                     "Africa/Addis_Ababa",             // UTC +03:00
	"FK" /* Falkland Islands (Malvinas) */ :                  "Atlantic/Stanley",               // UTC -03:00
	"FO" /* Faroe Islands */ :                                "Atlantic/Faroe",                 // UTC
	"FJ" /* Fiji */ :                                         "Pacific/Fiji",                   // UTC +12:00
	"FI" /* Finland */ :                                      "Europe/Helsinki",                // UTC +02:00
	"FR" /* France */ :                                       "Europe/Paris",                   // UTC +01:00
	"GF" /* French Guiana */ :                                "America/Cayenne",                // UTC -03:00
	"PF" /* French Polynesia */ :                             "Pacific/Tahiti",                 // UTC -10:00
	"TF" /* French Southern Territories */ :                  "Indian/Kerguelen",               // UTC +05:00
	"GA" /* Gabon */ :                                        "Africa/Libreville",              // UTC +01:00
	"GM" /* Gambia */ :                                       "Africa/Banjul",                  // UTC
	"GE" /* Georgia */ :                                      "Asia/Tbilisi",                   // UTC +04:00
	"DE" /* Germany */ :                                      "Europe/Berlin",                  // UTC +01:00
	"GH" /* Ghana */ :                                        "Africa/Accra",                   // UTC
	"GI" /* Gibraltar */ :                                    "Europe/Gibraltar",               // UTC +01:00
	"GR" /* Greece */ :                                       "Europe/Athens",                  // UTC +02:00
	"GL" /* Greenland */ :                                    "America/Nuuk",                   // UTC -02:00
	"GD" /* Grenada */ :                                      "America/Grenada",                // UTC -04:00
	"GP" /* Guadeloupe */ :                                   "America/Guadeloupe",             // UTC -04:00
	"GU" /* Guam */ :                                         "Pacific/Guam",                   // UTC +10:00
	"GT" /* Guatemala */ :                                    "America/Guatemala",              // UTC -06:00
	"GG" /* Guernsey */ :                                     "Europe/Guernsey",                // UTC
	"GN" /* Guinea */ :                                       "Africa/Conakry",                 // UTC
	"GW" /* Guinea-Bissau */ :                                "Africa/Bissau",                  // UTC
	"GY" /* Guyana */ :                                       "America/Guyana",                 // UTC -04:00
	"HT" /* Haiti */ :                                        "America/Port-au-Prince",         // UTC -05:00
	"VA" /* Holy See (Vatican City State) */ :                "Europe/Vatican",                 // UTC +01:00
	"HN" /* Honduras */ :                                     "America/Tegucigalpa",            // UTC -06:00
	"HK" /* Hong Kong */ :                                    "Asia/Hong_Kong",                 // UTC +08:00
	"HU" /* Hungary */ :                                      "Europe/Budapest",                // UTC +01:00
	"IS" /* Iceland */ :                                      "Atlantic/Reykjavik",             // UTC
	"IN" /* India */ :                                        "Asia/Kolkata",                   // UTC +05:30
	"ID" /* Indonesia */ :                                    "Asia/Jakarta",                   // UTC +07:00
	"ID-KS" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-KI" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-KU" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-SA" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-GO" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-ST" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-SR" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-SN" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-SG" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-BA" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-NB" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-NT" /* Indonesia */ :                                 "Asia/Makassar",                  // UTC +08:00
	"ID-MA" /* Indonesia */ :                                 "Asia/Jayapura",                  // UTC +09:00
	"ID-MU" /* Indonesia */ :                                 "Asia/Jayapura",                  // UTC +09:00
	"ID-PA" /* Indonesia */ :                                 "Asia/Jayapura",                  // UTC +09:00
	"ID-PB" /* Indonesia */ :                                 "Asia/Jayapura",                  // UTC +09:00
	"ID-PT" /* Indonesia */ :                                 "Asia/Jayapura",                  // UTC +09:00
	"ID-PE" /* Indonesia */ :                                 "Asia/Jayapura",                  // UTC +09:00
	"ID-PS" /* Indonesia */ :                                 "Asia/Jayapura",                  // UTC +09:00
	"ID-PD" /* Indonesia */ :                                 "Asia/Jayapura",                  // UTC +09:00
	"ID-KB" /* Indonesia */ :                                 "Asia/Pontianak",                 // UTC +07:00
	"IR" /* Iran, Islamic Republic of */ :                    "Asia/Tehran",                    // UTC +03:30
	"IQ" /* Iraq */ :                                         "Asia/Baghdad",                   // UTC +03:00
	"IE" /* Ireland */ :                                      "Europe/Dublin",                  // UTC
	"IM" /* Isle of Man */ :                                  "Europe/Isle_of_Man",             // UTC
	"IL" /* Israel */ :                                       "Asia/Jerusalem",                 // UTC +02:00
	"IT" /* Italy */ :                                        "Europe/Rome",                    // UTC +01:00
	"JM" /* Jamaica */ :                                      "America/Jamaica",                // UTC -05:00
	"JP" /* Japan */ :                                        "Asia/Tokyo",                     // UTC +09:00
	"JE" /* Jersey */ :                                       "Europe/Jersey",                  // UTC
	"JO" /* Jordan */ :                                       "Asia/Amman",                     // UTC +03:00
	"KZ" /* Kazakhstan */ :                                   "Asia/Almaty",                    // UTC +06:00
	"KE" /* Kenya */ :                                        "Africa/Nairobi",                 // UTC +03:00
	"KI" /* Kiribati */ :                                     "Pacific/Tarawa",                 // UTC +12:00
	"KP" /* Korea, Democratic People's Republic of */ :       "Asia/Pyongyang",                 // UTC +09:00
	"KR" /* Korea, Republic of */ :                           "Asia/Seoul",                     // UTC +09:00
	"KW" /* Kuwait */ :                                       "Asia/Kuwait",                    // UTC +03:00
	"KG" /* Kyrgyzstan */ :                                   "Asia/Bishkek",                   // UTC +06:00
	"LA" /* Lao People's Democratic Republic */ :             "Asia/Vientiane",                 // UTC +07:00
	"LV" /* Latvia */ :                                       "Europe/Riga",                    // UTC +02:00
	"LB" /* Lebanon */ :                                      "Asia/Beirut",                    // UTC +02:00
	"LS" /* Lesotho */ :                                      "Africa/Maseru",                  // UTC +02:00
	"LR" /* Liberia */ :                                      "Africa/Monrovia",                // UTC
	"LY" /* Libya */ :                                        "Africa/Tripoli",                 // UTC +02:00
	"LI" /* Liechtenstein */ :                                "Europe/Vaduz",                   // UTC +01:00
	"LT" /* Lithuania */ :                                    "Europe/Vilnius",                 // UTC +02:00
	"LU" /* Luxembourg */ :                                   "Europe/Luxembourg",              // UTC +01:00
	"MO" /* Macao */ :                                        "Asia/Macau",                     // UTC +08:00
	"MK" /* Macedonia, the Former Yugoslav Republic of */ :   "Europe/Skopje",                  // UTC +01:00
	"MG" /* Madagascar */ :                                   "Indian/Antananarivo",            // UTC +03:00
	"MW" /* Malawi */ :                                       "Africa/Blantyre",                // UTC +02:00
	"MY" /* Malaysia */ :                                     "Asia/Kuala_Lumpur",              // UTC +08:00
	"MV" /* Maldives */ :                                     "Indian/Maldives",                // UTC +05:00
	"ML" /* Mali */ :                                         "Africa/Bamako",                  // UTC
	"MT" /* Malta */ :                                        "Europe/Malta",                   // UTC +01:00
	"MH" /* Marshall Islands */ :                             "Pacific/Majuro",                 // UTC +12:00
	"MQ" /* Martinique */ :                                   "America/Martinique",             // UTC -04:00
	"MR" /* Mauritania */ :                                   "Africa/Nouakchott",              // UTC
	"MU" /* Mauritius */ :                                    "Indian/Mauritius",               // UTC +04:00
	"YT" /* Mayotte */ :                                      "Indian/Mayotte",                 // UTC +03:00
	"MX" /* Mexico */ :                                       "America/Mexico_City",            // UTC -06:00
	"MX-BN" /* Mexico */ :                                    "America/Tijuana",                // UTC -08:00
	"MX-QI" /* Mexico */ :                                    "America/Cancun",                 // UTC -05:00
	"MX-BS" /* Mexico */ :                                    "America/Mazatlan",               // UTC -07:00
	"MX-NA" /* Mexico */ :                                    "America/Mazatlan",               // UTC -07:00
	"MX-SI" /* Mexico */ :                                    "America/Mazatlan",               // UTC -07:00
	"MX-SO" /* Mexico */ :                                    "America/Hermosillo",             // UTC -07:00
	"MX-YU" /* Mexico */ :                                    "America/Merida",                 // UTC -06:00
	"MX-NL" /* Mexico */ :                                    "America/Monterrey",              // UTC -06:00
	"MX-TA" /* Mexico */ :                                    "America/Matamoros",              // UTC -06:00
	"MX-CI" /* Mexico */ :                                    "America/Chihuahua",              // UTC -06:00
	"FM" /* Micronesia, Federated States of */ :              "Pacific/Pohnpei",                // UTC +11:00
	"MD" /* Moldova, Republic of */ :                         "Europe/Chisinau",                // UTC +02:00
	"MC" /* Monaco */ :                                       "Europe/Monaco",                  // UTC +01:00
	"MN" /* Mongolia */ :                                     "Asia/Ulaanbaatar",               // UTC +08:00
	"ME" /* Montenegro */ :                                   "Europe/Podgorica",               // UTC +01:00
	"MS" /* Montserrat */ :                                   "America/Montserrat",             // UTC -04:00
	"MA" /* Morocco */ :                                      "Africa/Casablanca",              // UTC +01:00
	"MZ" /* Mozambique */ :                                   "Africa/Maputo",                  // UTC +02:00
	"MM" /* Myanmar */ :                                      "Asia/Yangon",                    // UTC +06:30
	"NA" /* Namibia */ :                                      "Africa/Windhoek",                // UTC +02:00
	"NR" /* Nauru */ :                                        "Pacific/Nauru",                  // UTC +12:00
	"NP" /* Nepal */ :                                        "Asia/Kathmandu",                 // UTC +05:45
	"NL" /* Netherlands */ :                                  "Europe/Amsterdam",               // UTC +01:00
	"NC" /* New Caledonia */ :                                "Pacific/Noumea",                 // UTC +11:00
	"NZ" /* New Zealand */ :                                  "Pacific/Auckland",               // UTC +13:00
	"NI" /* Nicaragua */ :                                    "America/Managua",                // UTC -06:00
	"NE" /* Niger */ :                                        "Africa/Niamey",                  // UTC +01:00
	"NG" /* Nigeria */ :                                      "Africa/Lagos",                   // UTC +01:00
	"NU" /* Niue */ :                                         "Pacific/Niue",                   // UTC -11:00
	"NF" /* Norfolk Island */ :                               "Pacific/Norfolk",                // UTC +12:00
	"MP" /* Northern Mariana Islands */ :                     "Pacific/Saipan",                 // UTC +10:00
	"NO" /* Norway */ :                                       "Europe/Oslo",                    // UTC +01:00
	"OM" /* Oman */ :                                         "Asia/Muscat",                    // UTC +04:00
	"PK" /* Pakistan */ :                                     "Asia/Karachi",                   // UTC +05:00
	"PW" /* Palau */ :                                        "Pacific/Palau",                  // UTC +09:00
	"PS" /* Palestine, State of */ :                          "Asia/Hebron",                    // UTC +02:00
	"PA" /* Panama */ :                                       "America/Panama",                 // UTC -05:00
	"PG" /* Papua New Guinea */ :                             "Pacific/Port_Moresby",           // UTC +10:00
	"PY" /* Paraguay */ :                                     "America/Asuncion",               // UTC -03:00
	"PE" /* Peru */ :                                         "America/Lima",                   // UTC -05:00
	"PH" /* Philippines */ :                                  "Asia/Manila",                    // UTC +08:00
	"PN" /* Pitcairn */ :                                     "Pacific/Pitcairn",               // UTC -08:00
	"PL" /* Poland */ :                                       "Europe/Warsaw",                  // UTC +01:00
	"PT" /* Portugal */ :                                     "Europe/Lisbon",                  // UTC
	"PR" /* Puerto Rico */ :                                  "America/Puerto_Rico",            // UTC -04:00
	"QA" /* Qatar */ :                                        "Asia/Qatar",                     // UTC +03:00
	"RO" /* Romania */ :                                      "Europe/Bucharest",               // UTC +02:00
	"RU" /* Russian Federation */ :                           "Europe/Moscow",                  // UTC +03:00
	"RU-KGD" /* Russian Federation */ :                       "Europe/Kaliningrad",             // UTC +02:00
	"RU-AST" /* Russian Federation */ :                       "Europe/Astrakhan",               // UTC +04:00
	"RU-SAM" /* Russian Federation */ :                       "Europe/Samara",                  // UTC +04:00
	"RU-SAR" /* Russian Federation */ :                       "Europe/Saratov",                 // UTC +04:00
	"RU-UD" /* Russian Federation */ :                        "Europe/Samara",                  // UTC +04:00
	"RU-ULY" /* Russian Federation */ :                       "Europe/Ulyanovsk",               // UTC +04:00
	"RU-BA" /* Russian Federation */ :                        "Asia/Yekaterinburg",             // UTC +05:00
	"RU-CHE" /* Russian Federation */ :                       "Asia/Yekaterinburg",             // UTC +05:00
	"RU-KHM" /* Russian Federation */ :                       "Asia/Yekaterinburg",             // UTC +05:00
	"RU-KGN" /* Russian Federation */ :                       "Asia/Yekaterinburg",             // UTC +05:00
	"RU-ORE" /* Russian Federation */ :                       "Asia/Yekaterinburg",             // UTC +05:00
	"RU-PER" /* Russian Federation */ :                       "Asia/Yekaterinburg",             // UTC +05:00
	"RU-SVE" /* Russian Federation */ :                       "Asia/Yekaterinburg",             // UTC +05:00
	"RU-TYU" /* Russian Federation */ :                       "Asia/Yekaterinburg",             // UTC +05:00
	"RU-YAN" /* Russian Federation */ :                       "Asia/Yekaterinburg",             // UTC +05:00
	"RU-OMS" /* Russian Federation */ :                       "Asia/Omsk",                      // UTC +06:00
	"RU-ALT" /* Russian Federation */ :                       "Asia/Barnaul",                   // UTC +07:00
	"RU-AL" /* Russian Federation */ :                        "Asia/Krasnoyarsk",               // UTC +07:00
	"RU-KEM" /* Russian Federation */ :                       "Asia/Novokuznetsk",              // UTC +07:00
	"RU-KK" /* Russian Federation */ :                        "Asia/Krasnoyarsk",               // UTC +07:00
	"RU-KYA" /* Russian Federation */ :                       "Asia/Krasnoyarsk",               // UTC +07:00
	"RU-NVS" /* Russian Federation */ :                       "Asia/Novosibirsk",               // UTC +07:00
	"RU-TOM" /* Russian Federation */ :                       "Asia/Tomsk",                     // UTC +07:00
	"RU-TY" /* Russian Federation */ :                        "Asia/Krasnoyarsk",               // UTC +07:00
	"RU-IRK" /* Russian Federation */ :                       "Asia/Irkutsk",                   // UTC +08:00
	"RU-BU" /* Russian Federation */ :                        "Asia/Irkutsk",                   // UTC +08:00
	"RU-AMU" /* Russian Federation */ :                       "Asia/Yakutsk",                   // UTC +09:00
	"RU-ZAB" /* Russian Federation */ :                       "Asia/Chita",                     // UTC +09:00
	"RU-SA" /* Russian Federation */ :                        "Asia/Khandyga",                  // UTC +09:00
	"RU-YEV" /* Russian Federation */ :                       "Asia/Vladivostok",               // UTC +10:00
	"RU-KHA" /* Russian Federation */ :                       "Asia/Vladivostok",               // UTC +10:00
	"RU-PRI" /* Russian Federation */ :                       "Asia/Vladivostok",               // UTC +10:00
	"RU-MAG" /* Russian Federation */ :                       "Asia/Magadan",                   // UTC +11:00
	"RU-SAK" /* Russian Federation */ :                       "Asia/Sakhalin",                  // UTC +11:00
	"RU-CHU" /* Russian Federation */ :                       "Asia/Anadyr",                    // UTC +12:00
	"RU-KAM" /* Russian Federation */ :                       "Asia/Kamchatka",                 // UTC +12:00
	"RU-VGG" /* Russian Federation */ :                       "Europe/Volgograd",               // UTC +03:00
	"RU-KIR" /* Russian Federation */ :                       "Europe/Kirov",                   // UTC +03:00
	"RW" /* Rwanda */ :                                       "Africa/Kigali",                  // UTC +02:00
	"RE" /* Réunion */ :                                      "Indian/Reunion",                 // UTC +04:00
	"BL" /* Saint Barthélemy */ :                             "America/St_Barthelemy",          // UTC -04:00
	"SH" /* Saint Helena, Ascension and Tristan da Cunha */ : "Atlantic/St_Helena",             // UTC
	"KN" /* Saint Kitts and Nevis */ :                        "America/St_Kitts",               // UTC -04:00
	"LC" /* Saint Lucia */ :                                  "America/St_Lucia",               // UTC -04:00
	"MF" /* Saint Martin (French part) */ :                   "America/Marigot",                // UTC -04:00
	"PM" /* Saint Pierre and Miquelon */ :                    "America/Miquelon",               // UTC -03:00
	"VC" /* Saint Vincent and the Grenadines */ :             "America/St_Vincent",             // UTC -04:00
	"WS" /* Samoa */ :                                        "Pacific/Apia",                   // UTC +13:00
	"SM" /* San Marino */ :                                   "Europe/San_Marino",              // UTC +01:00
	"ST" /* Sao Tome and Principe */ :                        "Africa/Sao_Tome",                // UTC
	"SA" /* Saudi Arabia */ :                                 "Asia/Riyadh",                    // UTC +03:00
	"SN" /* Senegal */ :                                      "Africa/Dakar",                   // UTC
	"RS" /* Serbia */ :                                       "Europe/Belgrade",                // UTC +01:00
	"SC" /* Seychelles */ :                                   "Indian/Mahe",                    // UTC +04:00
	"SL" /* Sierra Leone */ :                                 "Africa/Freetown",                // UTC
	"SG" /* Singapore */ :                                    "Asia/Singapore",                 // UTC +08:00
	"SX" /* Sint Maarten (Dutch part) */ :                    "America/Lower_Princes",          // UTC -04:00
	"SK" /* Slovakia */ :                                     "Europe/Bratislava",              // UTC +01:00
	"SI" /* Slovenia */ :                                     "Europe/Ljubljana",               // UTC +01:00
	"SB" /* Solomon Islands */ :                              "Pacific/Guadalcanal",            // UTC +11:00
	"SO" /* Somalia */ :                                      "Africa/Mogadishu",               // UTC +03:00
	"ZA" /* South Africa */ :                                 "Africa/Johannesburg",            // UTC +02:00
	"GS" /* South Georgia and the South Sandwich Islands */ : "Atlantic/South_Georgia",         // UTC -02:00
	"SS" /* South Sudan */ :                                  "Africa/Juba",                    // UTC +02:00
	"ES" /* Spain */ :                                        "Europe/Madrid",                  // UTC +01:00
	"LK" /* Sri Lanka */ :                                    "Asia/Colombo",                   // UTC +05:30
	"SD" /* Sudan */ :                                        "Africa/Khartoum",                // UTC +02:00
	"SR" /* Suriname */ :                                     "America/Paramaribo",             // UTC -03:00
	"SJ" /* Svalbard and Jan Mayen */ :                       "Arctic/Longyearbyen",            // UTC +01:00
	"SZ" /* Swaziland */ :                                    "Africa/Mbabane",                 // UTC +02:00
	"SE" /* Sweden */ :                                       "Europe/Stockholm",               // UTC +01:00
	"CH" /* Switzerland */ :                                  "Europe/Zurich",                  // UTC +01:00
	"SY" /* Syrian Arab Republic */ :                         "Asia/Damascus",                  // UTC +03:00
	"TW" /* Taiwan, Province of China */ :                    "Asia/Taipei",                    // UTC +08:00
	"TJ" /* Tajikistan */ :                                   "Asia/Dushanbe",                  // UTC +05:00
	"TZ" /* Tanzania, United Republic of */ :                 "Africa/Dar_es_Salaam",           // UTC +03:00
	"TH" /* Thailand */ :                                     "Asia/Bangkok",                   // UTC +07:00
	"TL" /* Timor-Leste */ :                                  "Asia/Dili",                      // UTC +09:00
	"TG" /* Togo */ :                                         "Africa/Lome",                    // UTC
	"TK" /* Tokelau */ :                                      "Pacific/Fakaofo",                // UTC +13:00
	"TO" /* Tonga */ :                                        "Pacific/Tongatapu",              // UTC +13:00
	"TT" /* Trinidad and Tobago */ :                          "America/Port_of_Spain",          // UTC -04:00
	"TN" /* Tunisia */ :                                      "Africa/Tunis",                   // UTC +01:00
	"TR" /* Turkey */ :                                       "Europe/Istanbul",                // UTC +03:00
	"TM" /* Turkmenistan */ :                                 "Asia/Ashgabat",                  // UTC +05:00
	"TC" /* Turks and Caicos Islands */ :                     "America/Grand_Turk",             // UTC -05:00
	"TV" /* Tuvalu */ :                                       "Pacific/Funafuti",               // UTC +12:00
	"UG" /* Uganda */ :                                       "Africa/Kampala",                 // UTC +03:00
	"UA" /* Ukraine */ :                                      "Europe/Kyiv",                    // UTC +02:00
	"AE" /* United Arab Emirates */ :                         "Asia/Dubai",                     // UTC +04:00
	"GB" /* United Kingdom */ :                               "Europe/London",                  // UTC
	"US" /* United States */ :                                "America/New_York",               // UTC -05:00
	"US-AL" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-AK" /* United States */ :                             "America/Anchorage",              // UTC -09:00
	"US-AZ" /* United States */ :                             "America/Phoenix",                // UTC -07:00
	"US-AR" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-CA" /* United States */ :                             "America/Los_Angeles",            // UTC -08:00
	"US-CO" /* United States */ :                             "America/Denver",                 // UTC -07:00
	"US-CT" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-DC" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-DE" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-FL" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-GA" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-HI" /* United States */ :                             "Pacific/Honolulu",               // UTC -10:00
	"US-ID" /* United States */ :                             "America/Boise",                  // UTC -07:00
	"US-IL" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-IN" /* United States */ :                             "America/Indiana/Indianapolis",   // UTC -05:00
	"US-IA" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-KS" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-KY" /* United States */ :                             "America/Kentucky/Louisville",    // UTC -06:00
	"US-LA" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-ME" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-MD" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-MA" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-MI" /* United States */ :                             "America/Detroit",                // UTC -05:00
	"US-MN" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-MS" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-MO" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-MT" /* United States */ :                             "America/Denver",                 // UTC -07:00
	"US-NE" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-NV" /* United States */ :                             "America/Los_Angeles",            // UTC -08:00
	"US-NH" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-NJ" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-NM" /* United States */ :                             "America/Denver",                 // UTC -07:00
	"US-NY" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-NC" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-ND" /* United States */ :                             "America/North_Dakota/Center",    // UTC -06:00
	"US-OH" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-OK" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-OR" /* United States */ :                             "America/Los_Angeles",            // UTC -08:00
	"US-PA" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-RI" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-SC" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-SD" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-TN" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-TX" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-UT" /* United States */ :                             "America/Denver",                 // UTC -07:00
	"US-VT" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-VA" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-WA" /* United States */ :                             "America/Los_Angeles",            // UTC -08:00
	"US-WV" /* United States */ :                             "America/New_York",               // UTC -05:00
	"US-WI" /* United States */ :                             "America/Chicago",                // UTC -06:00
	"US-WY" /* United States */ :                             "America/Denver",                 // UTC -07:00
	"UM" /* United States Minor Outlying Islands */ :         "Pacific/Midway",                 // UTC -11:00
	"UY" /* Uruguay */ :                                      "America/Montevideo",             // UTC -03:00
	"UZ" /* Uzbekistan */ :                                   "Asia/Tashkent",                  // UTC +05:00
	"VU" /* Vanuatu */ :                                      "Pacific/Efate",                  // UTC +11:00
	"VE" /* Venezuela, Bolivarian Republic of */ :            "America/Caracas",                // UTC -04:00
	"VN" /* Viet Nam */ :                                     "Asia/Ho_Chi_Minh",               // UTC +07:00
	"VG" /* Virgin Islands, British */ :                      "America/Tortola",                // UTC -04:00
	"VI" /* Virgin Islands, U.S. */ :                         "America/St_Thomas",              // UTC -04:00
	"WF" /* Wallis and Futuna */ :                            "Pacific/Wallis",                 // UTC +12:00
	"YE" /* Yemen */ :                                        "Asia/Aden",                      // UTC +03:00
	"ZM" /* Zambia */ :                                       "Africa/Lusaka",                  // UTC +02:00
	"ZW" /* Zimbabwe */ :                                     "Africa/Harare",                  // UTC +02:00
	"AX" /* Åland Islands */ :                                "Europe/Mariehamn",               // UTC +02:00
}

// TimeZoneOfGeography returns the time zone name of the given country and (optional) state.
// If one cannot be determined, the empty string is returned.
func TimeZoneOfGeography(country string, state string) (timeZone string) {
	if country == "" {
		return ""
	}
	country = strings.ToUpper(country)
	if state != "" {
		state = strings.ToUpper(state)
		if tz, ok := geoMap[country+"-"+state]; ok {
			return tz
		}
	}
	if tz, ok := geoMap[country]; ok {
		return tz
	}
	return ""
}
