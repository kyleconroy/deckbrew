package main

func TranslateID(id int) string {
	switch id {
	case 1071:
		return "Mishra's Factory (Fall)"
	case 1072:
		return "Mishra's Factory (Spring)"
	case 1073:
		return "Mishra's Factory (Summer)"
	case 1074:
		return "Mishra's Factory (Winter)"
	case 1076:
		return "Strip Mine (Even Horizon)"
	case 1077:
		return "Strip Mine (Uneven Horizon)"
	case 1078:
		return "Strip Mine (No Horizon)"
	case 1079:
		return "Strip Mine (Tower)"
	case 1080, 2888:
		return "Urza's Mine (Clawed Sphere)"
	case 1081, 2889:
		return "Urza's Mine (Mouth)"
	case 1082, 2890:
		return "Urza's Mine (Pulley)"
	case 1083, 2891:
		return "Urza's Mine (Tower)"
	case 1084, 2892:
		return "Urza's Power Plant (Bug)"
	case 1085, 2893:
		return "Urza's Power Plant (Columns)"
	case 1086, 2894:
		return "Urza's Power Plant (Sphere)"
	case 1087, 2895:
		return "Urza's Power Plant (Rock in Pot)"
	case 1088, 2896:
		return "Urza's Tower (Forest)"
	case 1089, 2897:
		return "Urza's Tower (Mountains)"
	case 2898, 1090:
		return "Urza's Tower (Plains)"
	case 1091, 2899:
		return "Urza's Tower (Shore)"
	case 4979:
		return "Pegasus Token"
	case 5472:
		return "Soldier Token"
	case 5503:
		return "Goblin Token"
	case 5560:
		return "Sheep Token"
	case 5601:
		return "Zombie Token"
	case 5607:
		return "Squirrel Token"
	case 9757:
		return "The Ultimate Nightmare of Wizards of the Coast Cu"
	case 9780:
		return "B.F.M. (Big Furry Monster Left)"
	case 9844:
		return "B.F.M. (Big Furry Monster Right)"
	case 74237:
		return "Our Market Research..."
	case 78968:
		return "Brothers Yamazaki (160a Sword)"
	case 85106:
		return "Brothers Yamazaki (160b Pike)"
	case 209163:
		return "Hornet Token"
	case 386322:
		return "Goblin Token"
	}
	return ""
}

func TCGSet(setId, set string) string {
	switch setId {
	case "10E":
		return "10th Edition"
	case "9ED":
		return "9th Edition"
	case "8ED":
		return "8th Edition"
	case "7ED":
		return "7th Edition"
	case "M15":
		return "Magic 2015 (M15)"
	case "M14":
		return "Magic 2014 (M14)"
	case "M13":
		return "Magic 2013 (M13)"
	case "M12":
		return "Magic 2012 (M12)"
	case "M11":
		return "Magic 2011 (M11)"
	case "M10":
		return "Magic 2010 (M10)"
	case "CMD":
		return "Commander"
	case "HHO":
		return "Special Occasion"
	case "RAV":
		return "Ravnica"
	case "DDG":
		return "Duel Decks: Knights vs Dragons"
	case "DDL":
		return "Duel Decks: Heroes vs. Monsters"
	case "PC2":
		return "Planechase 2012"
	case "C13":
		return "Commander 2013"
	case "C14":
		return "Commander 2014"
	case "PD2":
		return "Premium Deck Series: Fire and Lightning"
	case "LEB":
		return "Beta Edition"
	case "LEA":
		return "Alpha Edition"
	case "TSB":
		return "Timeshifted"
	case "MD1":
		return "Magic Modern Event Deck"
	case "CNS":
		return "Conspiracy"
	case "DKM":
		return "Deckmasters Garfield vs. Finkel"
	case "KTK":
		return "Khans of Tarkir"
	}
	return set
}
