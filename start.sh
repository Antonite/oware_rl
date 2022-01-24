for i in {1..5}
do
   echo "Starting learner $i..."
   ./oware_rl.exe > logs/log$i.txt 2>&1 & disown
done
