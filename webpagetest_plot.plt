reset
set datafile separator ","
set xdata time
set format x "%s"
set title "webpagetest for google search 'mozilla foundation'"
set xlabel "Date" 
set ylabel "SpeedIndex"
unset key
set xtics format "%b %d"
set grid

plot 'a' using 1:3
