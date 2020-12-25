qmmp &

sleep 3

PID="$(pgrep qmmp)"

while [ "$PID" = '' ]; do

sleep 1

PID="$(pgrep qmmp)"

done

xdotool windowactivate $(xdotool search --pid "$PID" | tail -1) ; sleep .1; xdotool getactivewindow windowmove 0 1144
