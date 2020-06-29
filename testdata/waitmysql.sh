#
echo Starting MySQL...

status="1"
while [ "$status" != "0" ]; do
  sleep 5
  mysql -e 'select VERSION()'
  status=$?
done

mysql -e 'select VERSION()'
