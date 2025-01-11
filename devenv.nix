{ pkgs, lib, config, inputs, ... }:
let
  pkgs-unstable = import inputs.nixpkgs-unstable { system = pkgs.stdenv.system; };
in
{
  # https://devenv.sh/basics/
  env.GREET = "clyper devenv";

  # https://devenv.sh/packages/
  packages = [ pkgs.git ];

  languages = {
    go = {
      enable = true;
      enableHardeningWorkaround = true;
      package = pkgs-unstable.go;
    };

    javascript = {
      enable = true;
    };
  };

  scripts.hello.exec = ''
    echo hello from $GREET
  '';

  scripts.backend.exec = ''
    go run --tags fts5 ./apps/clyper serve \
      --objstore ./testing/output2 \
      --fonts-dir ./testing/fonts \
      --font-name Freight
  '';

  enterShell = ''
    hello
    git --version
  '';

  # https://devenv.sh/tasks/
  # tasks = {
  #   "myproj:setup".exec = "mytool build";
  #   "devenv:enterShell".after = [ "myproj:setup" ];
  # };

  # https://devenv.sh/tests/
  enterTest = ''
    echo "Running tests"
    git --version | grep --color=auto "${pkgs.git.version}"
  '';

  # https://devenv.sh/pre-commit-hooks/
  # pre-commit.hooks.shellcheck.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
