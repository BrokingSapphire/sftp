"use client";

/**
 * Lightweight, dependency-free internationalisation for the app.
 *
 * - Locale is stored in localStorage and applied instantly (no route change).
 * - `t(key)` returns the string for the active locale, falling back to English,
 *   then to the key itself — so a missing translation never breaks the UI.
 * - Detailed coverage for the major languages used in India, plus English.
 */
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";

export const LOCALES = [
  { code: "en", label: "English", native: "English" },
  { code: "hi", label: "Hindi", native: "हिन्दी" },
  { code: "bn", label: "Bengali", native: "বাংলা" },
  { code: "ta", label: "Tamil", native: "தமிழ்" },
  { code: "te", label: "Telugu", native: "తెలుగు" },
  { code: "mr", label: "Marathi", native: "मराठी" },
  { code: "gu", label: "Gujarati", native: "ગુજરાતી" },
  { code: "kn", label: "Kannada", native: "ಕನ್ನಡ" },
  { code: "ml", label: "Malayalam", native: "മലയാളം" },
  { code: "pa", label: "Punjabi", native: "ਪੰਜਾਬੀ" },
  { code: "or", label: "Odia", native: "ଓଡ଼ିଆ" },
  { code: "as", label: "Assamese", native: "অসমীয়া" },
  { code: "ur", label: "Urdu", native: "اردو" },
] as const;

export type LocaleCode = (typeof LOCALES)[number]["code"];
export const RTL_LOCALES: LocaleCode[] = ["ur"];

// Translation keys. English is the source of truth; other locales fall back here.
export type TKey = keyof typeof en;

const en = {
  // Navigation
  "nav.dashboard": "Dashboard",
  "nav.files": "My Files",
  "nav.askAI": "Ask AI",
  "nav.teams": "Teams",
  "nav.common": "Common",
  "nav.shared_with_me": "Shared with me",
  "nav.inherited": "Inherited",
  "nav.starred": "Starred",
  "nav.shared": "Shared",
  "nav.trash": "Trash",
  "nav.administration": "Administration",
  "nav.users": "Users",
  "nav.storage": "Storage",
  "nav.audit": "Audit Log",
  "nav.security": "Security",
  "nav.backup": "Backup",
  "nav.apiKeys": "API Keys",
  // Actions
  "action.upload": "Upload",
  "action.newFolder": "New folder",
  "action.uploadFolder": "Upload folder",
  "action.search": "Search files…",
  "action.download": "Download",
  "action.delete": "Delete",
  "action.rename": "Rename",
  "action.share": "Share",
  "action.move": "Move",
  "action.copy": "Copy",
  "action.cancel": "Cancel",
  "action.save": "Save",
  "action.create": "Create",
  "action.logout": "Log out",
  "action.settings": "Settings",
  // Columns / common
  "col.name": "Name",
  "col.size": "Size",
  "col.modified": "Modified",
  "common.loading": "Loading…",
  "common.language": "Language",
  "common.welcomeBack": "Welcome back",
  "common.signIn": "Sign in",
  "common.home": "Home",
} as const;

// Only the most-used strings are translated per language; everything else falls
// back to English automatically. Keys omitted here inherit the English value.
type Dict = Partial<Record<TKey, string>>;

const hi: Dict = {
  "nav.dashboard": "डैशबोर्ड", "nav.files": "मेरी फ़ाइलें", "nav.askAI": "एआई से पूछें", "nav.teams": "टीमें",
  "nav.common": "साझा", "nav.shared_with_me": "मेरे साथ साझा", "nav.inherited": "प्राप्त", "nav.starred": "तारांकित",
  "nav.shared": "साझा किए गए", "nav.trash": "कूड़ेदान", "nav.administration": "प्रशासन", "nav.users": "उपयोगकर्ता",
  "nav.storage": "भंडारण", "nav.audit": "ऑडिट लॉग", "nav.security": "सुरक्षा", "nav.backup": "बैकअप", "nav.apiKeys": "एपीआई कुंजियाँ",
  "action.upload": "अपलोड करें", "action.newFolder": "नया फ़ोल्डर", "action.uploadFolder": "फ़ोल्डर अपलोड करें", "action.search": "फ़ाइलें खोजें…",
  "action.download": "डाउनलोड", "action.delete": "हटाएँ", "action.rename": "नाम बदलें", "action.share": "साझा करें",
  "action.move": "स्थानांतरित करें", "action.copy": "कॉपी करें", "action.cancel": "रद्द करें", "action.save": "सहेजें",
  "action.create": "बनाएँ", "action.logout": "लॉग आउट", "action.settings": "सेटिंग्स",
  "col.name": "नाम", "col.size": "आकार", "col.modified": "संशोधित", "common.loading": "लोड हो रहा है…",
  "common.language": "भाषा", "common.welcomeBack": "पुनः स्वागत है", "common.signIn": "साइन इन करें", "common.home": "होम",
};

const bn: Dict = {
  "nav.dashboard": "ড্যাশবোর্ড", "nav.files": "আমার ফাইল", "nav.askAI": "এআই-কে জিজ্ঞাসা করুন", "nav.teams": "দল",
  "nav.common": "সাধারণ", "nav.shared_with_me": "আমার সাথে শেয়ার করা", "nav.inherited": "প্রাপ্ত", "nav.starred": "তারাঙ্কিত",
  "nav.shared": "শেয়ার করা", "nav.trash": "ট্র্যাশ", "nav.administration": "প্রশাসন", "nav.users": "ব্যবহারকারী",
  "nav.storage": "স্টোরেজ", "nav.audit": "অডিট লগ", "nav.security": "নিরাপত্তা", "nav.backup": "ব্যাকআপ", "nav.apiKeys": "এপিআই কী",
  "action.upload": "আপলোড", "action.newFolder": "নতুন ফোল্ডার", "action.uploadFolder": "ফোল্ডার আপলোড", "action.search": "ফাইল খুঁজুন…",
  "action.download": "ডাউনলোড", "action.delete": "মুছুন", "action.rename": "নাম পরিবর্তন", "action.share": "শেয়ার",
  "action.cancel": "বাতিল", "action.save": "সংরক্ষণ", "action.create": "তৈরি করুন", "action.logout": "লগ আউট", "action.settings": "সেটিংস",
  "col.name": "নাম", "col.size": "আকার", "col.modified": "পরিবর্তিত", "common.loading": "লোড হচ্ছে…",
  "common.language": "ভাষা", "common.welcomeBack": "আবার স্বাগতম", "common.signIn": "সাইন ইন", "common.home": "হোম",
};

const ta: Dict = {
  "nav.dashboard": "டாஷ்போர்டு", "nav.files": "எனது கோப்புகள்", "nav.askAI": "AI-யிடம் கேளுங்கள்", "nav.teams": "குழுக்கள்",
  "nav.common": "பொது", "nav.shared_with_me": "என்னுடன் பகிரப்பட்டது", "nav.inherited": "பெறப்பட்டது", "nav.starred": "நட்சத்திரமிட்டது",
  "nav.shared": "பகிரப்பட்டது", "nav.trash": "குப்பை", "nav.administration": "நிர்வாகம்", "nav.users": "பயனர்கள்",
  "nav.storage": "சேமிப்பு", "nav.audit": "தணிக்கை பதிவு", "nav.security": "பாதுகாப்பு", "nav.backup": "காப்புப்பிரதி", "nav.apiKeys": "API விசைகள்",
  "action.upload": "பதிவேற்று", "action.newFolder": "புதிய கோப்புறை", "action.uploadFolder": "கோப்புறையைப் பதிவேற்று", "action.search": "கோப்புகளைத் தேடு…",
  "action.download": "பதிவிறக்கு", "action.delete": "நீக்கு", "action.rename": "பெயர் மாற்று", "action.share": "பகிர்",
  "action.cancel": "ரத்து", "action.save": "சேமி", "action.create": "உருவாக்கு", "action.logout": "வெளியேறு", "action.settings": "அமைப்புகள்",
  "col.name": "பெயர்", "col.size": "அளவு", "col.modified": "மாற்றப்பட்டது", "common.loading": "ஏற்றுகிறது…",
  "common.language": "மொழி", "common.welcomeBack": "மீண்டும் வரவேற்கிறோம்", "common.signIn": "உள்நுழை", "common.home": "முகப்பு",
};

const te: Dict = {
  "nav.dashboard": "డాష్‌బోర్డ్", "nav.files": "నా ఫైళ్లు", "nav.askAI": "AIని అడగండి", "nav.teams": "బృందాలు",
  "nav.common": "సాధారణ", "nav.shared_with_me": "నాతో భాగస్వామ్యం", "nav.inherited": "వారసత్వం", "nav.starred": "నక్షత్రం గుర్తు",
  "nav.shared": "భాగస్వామ్యం", "nav.trash": "చెత్త", "nav.administration": "నిర్వహణ", "nav.users": "వినియోగదారులు",
  "nav.storage": "నిల్వ", "nav.audit": "ఆడిట్ లాగ్", "nav.security": "భద్రత", "nav.backup": "బ్యాకప్", "nav.apiKeys": "API కీలు",
  "action.upload": "అప్‌లోడ్", "action.newFolder": "కొత్త ఫోల్డర్", "action.uploadFolder": "ఫోల్డర్ అప్‌లోడ్", "action.search": "ఫైళ్లను వెతకండి…",
  "action.download": "డౌన్‌లోడ్", "action.delete": "తొలగించు", "action.rename": "పేరు మార్చు", "action.share": "భాగస్వామ్యం",
  "action.cancel": "రద్దు", "action.save": "సేవ్", "action.create": "సృష్టించు", "action.logout": "లాగ్ అవుట్", "action.settings": "సెట్టింగ్‌లు",
  "col.name": "పేరు", "col.size": "పరిమాణం", "col.modified": "మార్చబడింది", "common.loading": "లోడ్ అవుతోంది…",
  "common.language": "భాష", "common.welcomeBack": "తిరిగి స్వాగతం", "common.signIn": "సైన్ ఇన్", "common.home": "హోమ్",
};

const mr: Dict = {
  "nav.dashboard": "डॅशबोर्ड", "nav.files": "माझ्या फायली", "nav.askAI": "एआयला विचारा", "nav.teams": "संघ",
  "nav.common": "सामायिक", "nav.shared_with_me": "माझ्यासोबत सामायिक", "nav.inherited": "प्राप्त", "nav.starred": "तारांकित",
  "nav.shared": "सामायिक केलेले", "nav.trash": "कचरा", "nav.administration": "प्रशासन", "nav.users": "वापरकर्ते",
  "nav.storage": "संचय", "nav.audit": "ऑडिट लॉग", "nav.security": "सुरक्षा", "nav.backup": "बॅकअप", "nav.apiKeys": "API कळा",
  "action.upload": "अपलोड", "action.newFolder": "नवीन फोल्डर", "action.uploadFolder": "फोल्डर अपलोड", "action.search": "फायली शोधा…",
  "action.download": "डाउनलोड", "action.delete": "हटवा", "action.rename": "नाव बदला", "action.share": "सामायिक करा",
  "action.cancel": "रद्द", "action.save": "जतन करा", "action.create": "तयार करा", "action.logout": "बाहेर पडा", "action.settings": "सेटिंग्ज",
  "col.name": "नाव", "col.size": "आकार", "col.modified": "सुधारित", "common.loading": "लोड होत आहे…",
  "common.language": "भाषा", "common.welcomeBack": "पुन्हा स्वागत आहे", "common.signIn": "साइन इन", "common.home": "मुख्यपृष्ठ",
};

const gu: Dict = {
  "nav.dashboard": "ડેશબોર્ડ", "nav.files": "મારી ફાઇલો", "nav.askAI": "AI ને પૂછો", "nav.teams": "ટીમો",
  "nav.common": "સામાન્ય", "nav.shared_with_me": "મારી સાથે શેર કરેલ", "nav.inherited": "પ્રાપ્ત", "nav.starred": "તારાંકિત",
  "nav.shared": "શેર કરેલ", "nav.trash": "કચરાપેટી", "nav.administration": "વહીવટ", "nav.users": "વપરાશકર્તાઓ",
  "nav.storage": "સંગ્રહ", "nav.audit": "ઑડિટ લૉગ", "nav.security": "સુરક્ષા", "nav.backup": "બેકઅપ", "nav.apiKeys": "API કીઓ",
  "action.upload": "અપલોડ", "action.newFolder": "નવું ફોલ્ડર", "action.uploadFolder": "ફોલ્ડર અપલોડ", "action.search": "ફાઇલો શોધો…",
  "action.download": "ડાઉનલોડ", "action.delete": "કાઢી નાખો", "action.rename": "નામ બદલો", "action.share": "શેર કરો",
  "action.cancel": "રદ કરો", "action.save": "સાચવો", "action.create": "બનાવો", "action.logout": "લૉગ આઉટ", "action.settings": "સેટિંગ્સ",
  "col.name": "નામ", "col.size": "કદ", "col.modified": "સંશોધિત", "common.loading": "લોડ થઈ રહ્યું છે…",
  "common.language": "ભાષા", "common.welcomeBack": "પાછા સ્વાગત છે", "common.signIn": "સાઇન ઇન", "common.home": "હોમ",
};

const kn: Dict = {
  "nav.dashboard": "ಡ್ಯಾಶ್‌ಬೋರ್ಡ್", "nav.files": "ನನ್ನ ಫೈಲ್‌ಗಳು", "nav.askAI": "AI ಅನ್ನು ಕೇಳಿ", "nav.teams": "ತಂಡಗಳು",
  "nav.common": "ಸಾಮಾನ್ಯ", "nav.shared_with_me": "ನನ್ನೊಂದಿಗೆ ಹಂಚಿಕೊಂಡಿದೆ", "nav.inherited": "ಪಡೆದಿದೆ", "nav.starred": "ನಕ್ಷತ್ರ ಗುರುತು",
  "nav.shared": "ಹಂಚಿಕೊಂಡಿದೆ", "nav.trash": "ಕಸ", "nav.administration": "ಆಡಳಿತ", "nav.users": "ಬಳಕೆದಾರರು",
  "nav.storage": "ಸಂಗ್ರಹಣೆ", "nav.audit": "ಆಡಿಟ್ ಲಾಗ್", "nav.security": "ಭದ್ರತೆ", "nav.backup": "ಬ್ಯಾಕಪ್", "nav.apiKeys": "API ಕೀಗಳು",
  "action.upload": "ಅಪ್‌ಲೋಡ್", "action.newFolder": "ಹೊಸ ಫೋಲ್ಡರ್", "action.uploadFolder": "ಫೋಲ್ಡರ್ ಅಪ್‌ಲೋಡ್", "action.search": "ಫೈಲ್‌ಗಳನ್ನು ಹುಡುಕಿ…",
  "action.download": "ಡೌನ್‌ಲೋಡ್", "action.delete": "ಅಳಿಸಿ", "action.rename": "ಮರುಹೆಸರಿಸಿ", "action.share": "ಹಂಚಿಕೊಳ್ಳಿ",
  "action.cancel": "ರದ್ದುಮಾಡಿ", "action.save": "ಉಳಿಸಿ", "action.create": "ರಚಿಸಿ", "action.logout": "ಲಾಗ್ ಔಟ್", "action.settings": "ಸೆಟ್ಟಿಂಗ್‌ಗಳು",
  "col.name": "ಹೆಸರು", "col.size": "ಗಾತ್ರ", "col.modified": "ಮಾರ್ಪಡಿಸಲಾಗಿದೆ", "common.loading": "ಲೋಡ್ ಆಗುತ್ತಿದೆ…",
  "common.language": "ಭಾಷೆ", "common.welcomeBack": "ಮರಳಿ ಸ್ವಾಗತ", "common.signIn": "ಸೈನ್ ಇನ್", "common.home": "ಮುಖಪುಟ",
};

const ml: Dict = {
  "nav.dashboard": "ഡാഷ്‌ബോർഡ്", "nav.files": "എന്റെ ഫയലുകൾ", "nav.askAI": "AI-യോട് ചോദിക്കുക", "nav.teams": "ടീമുകൾ",
  "nav.common": "പൊതുവായത്", "nav.shared_with_me": "എന്നോട് പങ്കിട്ടത്", "nav.inherited": "ലഭിച്ചത്", "nav.starred": "നക്ഷത്രമിട്ടത്",
  "nav.shared": "പങ്കിട്ടത്", "nav.trash": "ട്രാഷ്", "nav.administration": "ഭരണം", "nav.users": "ഉപയോക്താക്കൾ",
  "nav.storage": "സംഭരണം", "nav.audit": "ഓഡിറ്റ് ലോഗ്", "nav.security": "സുരക്ഷ", "nav.backup": "ബാക്കപ്പ്", "nav.apiKeys": "API കീകൾ",
  "action.upload": "അപ്‌ലോഡ്", "action.newFolder": "പുതിയ ഫോൾഡർ", "action.uploadFolder": "ഫോൾഡർ അപ്‌ലോഡ്", "action.search": "ഫയലുകൾ തിരയുക…",
  "action.download": "ഡൗൺലോഡ്", "action.delete": "ഇല്ലാതാക്കുക", "action.rename": "പേരുമാറ്റുക", "action.share": "പങ്കിടുക",
  "action.cancel": "റദ്ദാക്കുക", "action.save": "സംരക്ഷിക്കുക", "action.create": "സൃഷ്ടിക്കുക", "action.logout": "ലോഗ് ഔട്ട്", "action.settings": "ക്രമീകരണങ്ങൾ",
  "col.name": "പേര്", "col.size": "വലുപ്പം", "col.modified": "പരിഷ്കരിച്ചത്", "common.loading": "ലോഡ് ചെയ്യുന്നു…",
  "common.language": "ഭാഷ", "common.welcomeBack": "വീണ്ടും സ്വാഗതം", "common.signIn": "സൈൻ ഇൻ", "common.home": "ഹോം",
};

const pa: Dict = {
  "nav.dashboard": "ਡੈਸ਼ਬੋਰਡ", "nav.files": "ਮੇਰੀਆਂ ਫਾਈਲਾਂ", "nav.askAI": "AI ਨੂੰ ਪੁੱਛੋ", "nav.teams": "ਟੀਮਾਂ",
  "nav.common": "ਸਾਂਝਾ", "nav.shared_with_me": "ਮੇਰੇ ਨਾਲ ਸਾਂਝਾ", "nav.inherited": "ਪ੍ਰਾਪਤ", "nav.starred": "ਤਾਰਾ ਲੱਗਾ",
  "nav.shared": "ਸਾਂਝਾ ਕੀਤਾ", "nav.trash": "ਰੱਦੀ", "nav.administration": "ਪ੍ਰਸ਼ਾਸਨ", "nav.users": "ਵਰਤੋਂਕਾਰ",
  "nav.storage": "ਸਟੋਰੇਜ", "nav.audit": "ਆਡਿਟ ਲਾਗ", "nav.security": "ਸੁਰੱਖਿਆ", "nav.backup": "ਬੈਕਅੱਪ", "nav.apiKeys": "API ਕੁੰਜੀਆਂ",
  "action.upload": "ਅਪਲੋਡ", "action.newFolder": "ਨਵਾਂ ਫੋਲਡਰ", "action.uploadFolder": "ਫੋਲਡਰ ਅਪਲੋਡ", "action.search": "ਫਾਈਲਾਂ ਖੋਜੋ…",
  "action.download": "ਡਾਊਨਲੋਡ", "action.delete": "ਮਿਟਾਓ", "action.rename": "ਨਾਂ ਬਦਲੋ", "action.share": "ਸਾਂਝਾ ਕਰੋ",
  "action.cancel": "ਰੱਦ ਕਰੋ", "action.save": "ਸੰਭਾਲੋ", "action.create": "ਬਣਾਓ", "action.logout": "ਲਾਗ ਆਊਟ", "action.settings": "ਸੈਟਿੰਗਾਂ",
  "col.name": "ਨਾਂ", "col.size": "ਆਕਾਰ", "col.modified": "ਸੋਧਿਆ", "common.loading": "ਲੋਡ ਹੋ ਰਿਹਾ ਹੈ…",
  "common.language": "ਭਾਸ਼ਾ", "common.welcomeBack": "ਮੁੜ ਸੁਆਗਤ ਹੈ", "common.signIn": "ਸਾਈਨ ਇਨ", "common.home": "ਹੋਮ",
};

const or: Dict = {
  "nav.dashboard": "ଡ୍ୟାସବୋର୍ଡ", "nav.files": "ମୋର ଫାଇଲଗୁଡ଼ିକ", "nav.askAI": "AIକୁ ପଚାରନ୍ତୁ", "nav.teams": "ଦଳ",
  "nav.common": "ସାଧାରଣ", "nav.shared_with_me": "ମୋ ସହ ଅଂଶୀଦାର", "nav.inherited": "ପ୍ରାପ୍ତ", "nav.starred": "ତାରକାଙ୍କିତ",
  "nav.shared": "ଅଂଶୀଦାର", "nav.trash": "ଆବର୍ଜନା", "nav.administration": "ପ୍ରଶାସନ", "nav.users": "ବ୍ୟବହାରକାରୀ",
  "nav.storage": "ଭଣ୍ଡାର", "nav.audit": "ଅଡିଟ୍ ଲଗ୍", "nav.security": "ସୁରକ୍ଷା", "nav.backup": "ବ୍ୟାକଅପ୍", "nav.apiKeys": "API କୀ",
  "action.upload": "ଅପଲୋଡ୍", "action.newFolder": "ନୂଆ ଫୋଲ୍ଡର", "action.uploadFolder": "ଫୋଲ୍ଡର ଅପଲୋଡ୍", "action.search": "ଫାଇଲ ଖୋଜନ୍ତୁ…",
  "action.download": "ଡାଉନଲୋଡ୍", "action.delete": "ବିଲୋପ", "action.rename": "ନାମ ବଦଳାନ୍ତୁ", "action.share": "ଅଂଶୀଦାର",
  "action.cancel": "ବାତିଲ୍", "action.save": "ସଞ୍ଚୟ", "action.create": "ସୃଷ୍ଟି", "action.logout": "ଲଗ୍ ଆଉଟ୍", "action.settings": "ସେଟିଂସ୍",
  "col.name": "ନାମ", "col.size": "ଆକାର", "col.modified": "ପରିବର୍ତ୍ତିତ", "common.loading": "ଲୋଡ୍ ହେଉଛି…",
  "common.language": "ଭାଷା", "common.welcomeBack": "ପୁନଃ ସ୍ୱାଗତ", "common.signIn": "ସାଇନ୍ ଇନ୍", "common.home": "ହୋମ",
};

const as: Dict = {
  "nav.dashboard": "ডেশ্বব’ৰ্ড", "nav.files": "মোৰ ফাইল", "nav.askAI": "AIক সোধক", "nav.teams": "দল",
  "nav.common": "সাধাৰণ", "nav.shared_with_me": "মোৰ সৈতে ভাগ কৰা", "nav.inherited": "প্ৰাপ্ত", "nav.starred": "তৰাচিহ্নিত",
  "nav.shared": "ভাগ কৰা", "nav.trash": "আৱৰ্জনা", "nav.administration": "প্ৰশাসন", "nav.users": "ব্যৱহাৰকাৰী",
  "nav.storage": "সংৰক্ষণ", "nav.audit": "অডিট লগ", "nav.security": "সুৰক্ষা", "nav.backup": "বেকআপ", "nav.apiKeys": "API কী",
  "action.upload": "আপল’ড", "action.newFolder": "নতুন ফ'ল্ডাৰ", "action.uploadFolder": "ফ'ল্ডাৰ আপল’ড", "action.search": "ফাইল বিচাৰক…",
  "action.download": "ডাউনল’ড", "action.delete": "মচক", "action.rename": "নাম সলনি", "action.share": "ভাগ কৰক",
  "action.cancel": "বাতিল", "action.save": "সাঁচি থওক", "action.create": "সৃষ্টি কৰক", "action.logout": "লগ আউট", "action.settings": "ছেটিংছ",
  "col.name": "নাম", "col.size": "আকাৰ", "col.modified": "সংশোধিত", "common.loading": "ল’ড হৈ আছে…",
  "common.language": "ভাষা", "common.welcomeBack": "পুনৰ স্বাগতম", "common.signIn": "ছাইন ইন", "common.home": "হোম",
};

const ur: Dict = {
  "nav.dashboard": "ڈیش بورڈ", "nav.files": "میری فائلیں", "nav.askAI": "اے آئی سے پوچھیں", "nav.teams": "ٹیمیں",
  "nav.common": "مشترکہ", "nav.shared_with_me": "میرے ساتھ اشتراک", "nav.inherited": "موصول", "nav.starred": "ستارہ لگا",
  "nav.shared": "اشتراک شدہ", "nav.trash": "ردی", "nav.administration": "انتظامیہ", "nav.users": "صارفین",
  "nav.storage": "اسٹوریج", "nav.audit": "آڈٹ لاگ", "nav.security": "سیکیورٹی", "nav.backup": "بیک اپ", "nav.apiKeys": "اے پی آئی کیز",
  "action.upload": "اپ لوڈ", "action.newFolder": "نیا فولڈر", "action.uploadFolder": "فولڈر اپ لوڈ", "action.search": "فائلیں تلاش کریں…",
  "action.download": "ڈاؤن لوڈ", "action.delete": "حذف کریں", "action.rename": "نام بدلیں", "action.share": "اشتراک کریں",
  "action.cancel": "منسوخ", "action.save": "محفوظ کریں", "action.create": "بنائیں", "action.logout": "لاگ آؤٹ", "action.settings": "ترتیبات",
  "col.name": "نام", "col.size": "سائز", "col.modified": "ترمیم شدہ", "common.loading": "لوڈ ہو رہا ہے…",
  "common.language": "زبان", "common.welcomeBack": "دوبارہ خوش آمدید", "common.signIn": "سائن ان", "common.home": "ہوم",
};

const DICTS: Record<LocaleCode, Dict> = { en, hi, bn, ta, te, mr, gu, kn, ml, pa, or, as, ur };

interface I18nValue {
  locale: LocaleCode;
  setLocale: (l: LocaleCode) => void;
  t: (key: TKey) => string;
  dir: "ltr" | "rtl";
}

const I18nCtx = createContext<I18nValue | null>(null);
const STORAGE_KEY = "sphr_locale";

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<LocaleCode>("en");

  useEffect(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY) as LocaleCode | null;
      if (saved && DICTS[saved]) setLocaleState(saved);
    } catch { /* ignore */ }
  }, []);

  const setLocale = useCallback((l: LocaleCode) => {
    setLocaleState(l);
    try { localStorage.setItem(STORAGE_KEY, l); } catch { /* ignore */ }
    if (typeof document !== "undefined") {
      document.documentElement.lang = l;
      document.documentElement.dir = RTL_LOCALES.includes(l) ? "rtl" : "ltr";
    }
  }, []);

  useEffect(() => {
    if (typeof document !== "undefined") {
      document.documentElement.lang = locale;
      document.documentElement.dir = RTL_LOCALES.includes(locale) ? "rtl" : "ltr";
    }
  }, [locale]);

  const t = useCallback((key: TKey) => DICTS[locale]?.[key] ?? en[key] ?? key, [locale]);

  const value = useMemo<I18nValue>(
    () => ({ locale, setLocale, t, dir: RTL_LOCALES.includes(locale) ? "rtl" : "ltr" }),
    [locale, setLocale, t],
  );

  return <I18nCtx.Provider value={value}>{children}</I18nCtx.Provider>;
}

export function useI18n(): I18nValue {
  const ctx = useContext(I18nCtx);
  if (!ctx) return { locale: "en", setLocale: () => {}, t: (k) => en[k] ?? k, dir: "ltr" };
  return ctx;
}
