<?php

define("FOO", 1);
define('FOO', 2);
define('BAR', 3);

$uid = 10;
$pid = 3493;
var_dump($uid);
var_dump($pid);

class C {
  const BAZ = 100;
}

var_dump(FOO);
var_dump(BAR);
var_dump(C::BAZ);
