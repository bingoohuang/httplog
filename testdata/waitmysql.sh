while:
do
  sleep 5
  mysql -e 'select version()'
  if [ $? = 0 ]; then
    break
  fi
  echo "server logs"
  docker logs --tail 5 mysqld
done

mysql -e 'select VERSION()'

