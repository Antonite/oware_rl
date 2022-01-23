for i in {1..15}
do
   echo "Starting learner $i..."
   ./oware_rl.exe >> logs/logs.txt 2>&1 & disown
done
