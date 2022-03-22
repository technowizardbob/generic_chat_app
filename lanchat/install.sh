#!/bin/bash
file=lanchat
sh="$file".sh
icon=lanchat.png
yaml=conf.yaml
desktop=LanChat.desktop
installto=/opt/lanchat
app_exists() {
   if [[ -f "$1" && -r "$1" && -x "$1" ]]; then
	true;
   else
	false;
   fi
}
file_exists() {
   if [[ -f "$1" && -r "$1" ]]; then
	true;
   else
	false;
   fi
}
if [ ! "$EUID" -eq 0 ]; then
   echo "Please install me as Root!"
   exit 1
fi

if ! `app_exists "$file"`; then
   echo "Do an 'go build' first, so you have an exec to use called $file"
   exit 1
fi

echo "Installing..."
mkdir -p "$installto"
ln -s "$(pwd -P)"/"$file" "$installto"/

if `file_exists "$icon"`; then
   cp "$icon" "$installto"/
fi

if `file_exists "$yaml"`; then
   ln -s "$(pwd -P)"/"$yaml" "$installto"/
fi

if `file_exists "$desktop"`; then
   ln -s "$(pwd -P)"/"$desktop" "$installto"/
   ln -s "$installto"/"$desktop" /usr/share/applications/
fi

#echo "pushd $installto > /dev/null" > "$installto"/"$sh"
#echo "$installto/$file" >> "$installto"/"$sh"
#echo "popd > /dev/null" >> "$installto"/"$sh"
#chmod +x "$installto"/"$sh"
