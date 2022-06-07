gtimeout 150 go run . 0 > 1.txt &
gtimeout 150 go run . 1 > 2.txt &
gtimeout 150 go run . 2 > 3.txt &
gtimeout 20 go run . 3 > 4.txt &
