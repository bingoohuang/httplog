#
echo Starting MySQL...

status="1"
while [ "$status" != "0" ]; do
  sleep 1
  mysql -e 'select version()'
  status=$?
done

mysql -e 'select VERSION()'

