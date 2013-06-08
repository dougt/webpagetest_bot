reset
set term png size 1920, 1080
set output "output.png"

set datafile separator ","
set timefmt "%s"
set xdata time
set title "webpagetest for google search 'mozilla foundation'"
set xlabel "Date" 
set ylabel "SpeedIndex"
unset key
set grid

plot 'a' using 1:3
