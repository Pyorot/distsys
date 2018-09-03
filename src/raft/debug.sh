test=$1
passes=0
fails=0
for ((n=0; n<10; n++))
do
  output=$(go test -run $test -count=1)
  if (echo "$output" | grep -q PASS)
  then
    ((passes++))
  else
    echo "$output" > ../../results/"$test"-"$n".txt
    ((fails++))
  fi
done
echo "$test: $passes passes; $fails fails"