#!/usr/bin/env bash
# Build a minimal, dependency-free APK without Gradle (offline & deterministic).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SDK="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-$HOME/Android/Sdk}}"

# pick latest available build-tools
BT="$(ls -d "$SDK"/build-tools/*/ 2>/dev/null | sort -V | tail -1)"
BT="${BT%/}"
PLATFORM="$SDK/platforms/android-34/android.jar"

AAPT2="$BT/aapt2"
D8="$BT/d8"
ZIPALIGN="$BT/zipalign"
APKSIGNER="$BT/apksigner"

for t in "$AAPT2" "$D8" "$ZIPALIGN" "$APKSIGNER" "$PLATFORM"; do
  [ -e "$t" ] || { echo "missing: $t" >&2; exit 1; }
done

APPNAME="devmon"
PKG="com.qvacloud.devmon"
AND="$ROOT/android"
BUILD="$AND/build"
OUT="$ROOT/dist"
KS="$AND/debug.keystore"
KSPASS="android"
KSALIAS="androidkey"

rm -rf "$BUILD"
mkdir -p "$BUILD/classes" "$BUILD/dex" "$OUT"

echo ">> aapt2 link"
"$AAPT2" link \
  -o "$BUILD/base.apk" \
  -I "$PLATFORM" \
  --manifest "$AND/AndroidManifest.xml" \
  --min-sdk-version 30 \
  --target-sdk-version 34 \
  --version-code 1 \
  --version-name "1.0"

echo ">> javac"
find "$AND/java" -name '*.java' > "$BUILD/sources.txt"
javac -encoding UTF-8 -source 11 -target 11 -nowarn \
  -classpath "$PLATFORM" -d "$BUILD/classes" @"$BUILD/sources.txt"

echo ">> d8"
find "$BUILD/classes" -name '*.class' > "$BUILD/classes.txt"
"$D8" --lib "$PLATFORM" --min-api 30 --output "$BUILD/dex" @"$BUILD/classes.txt"

echo ">> package dex"
cp "$BUILD/base.apk" "$BUILD/unsigned.apk"
( cd "$BUILD/dex" && zip -q -X "$BUILD/unsigned.apk" classes.dex )

echo ">> zipalign"
"$ZIPALIGN" -f -p 4 "$BUILD/unsigned.apk" "$BUILD/aligned.apk"

if [ ! -f "$KS" ]; then
  echo ">> generate debug keystore"
  keytool -genkeypair -v -keystore "$KS" -alias "$KSALIAS" \
    -keyalg RSA -keysize 2048 -validity 10000 \
    -storepass "$KSPASS" -keypass "$KSPASS" \
    -dname "CN=devmon, OU=dev, O=dev, L=-, S=-, C=ES" >/dev/null 2>&1
fi

echo ">> sign"
"$APKSIGNER" sign \
  --ks "$KS" --ks-pass "pass:$KSPASS" --key-pass "pass:$KSPASS" --ks-key-alias "$KSALIAS" \
  --out "$OUT/$APPNAME.apk" "$BUILD/aligned.apk"

"$APKSIGNER" verify "$OUT/$APPNAME.apk" >/dev/null
echo "OK -> $OUT/$APPNAME.apk ($(du -h "$OUT/$APPNAME.apk" | cut -f1))"
