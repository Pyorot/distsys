# test=$1
passes=0
fails=0
for ((n=28; n<100; n++))
do
  output=$(go test -run TestReElection2A -count=1)
  # output=$(go test -run $test -count=1)
  if (echo "$output" | grep -q FAIL)
  then
    echo "$output" > ../../results/"$n".txt
    echo "$n - fail"
    ((fails++))
  else
    msg=$(echo "$output" | grep raft | sed "s/ok\s*raft\s*//")
    echo "$n - $msg"
    ((passes++))
  fi
done
echo "RESULT: $passes passes; $fails fails"