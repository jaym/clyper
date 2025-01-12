{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:
let
  pkgs-unstable = import inputs.nixpkgs-unstable { system = pkgs.stdenv.system; };
in
{
  # https://devenv.sh/basics/
  env.GREET = "clyper devenv";

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.ffmpeg
    pkgs.curl
  ];

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

  scripts.generate-test-video.exec =
    let
      testdata-dir = "testing/samplevideo";
    in
    ''
      set -ex
      mkdir -p ${testdata-dir}
      FNAME="Test.S01E01.mp4"
      FPATH="${testdata-dir}/$FNAME"
      TMPFPATH="${testdata-dir}/tmp.$FNAME"
      SUBTITLE_FILE="${testdata-dir}/subs.srt"
      ffmpeg -f lavfi -i color=c=red:duration=2:size=640x360 -f lavfi -i color=c=green:duration=2:size=640x360 \
        -f lavfi -i color=c=blue:duration=2:size=640x360 -f lavfi -i color=c=yellow:duration=2:size=640x360 \
        -filter_complex "[0:v:0][1:v:0][2:v:0][3:v:0]concat=n=4:v=1[outv]" -map "[outv]" $TMPFPATH
      cat <<EOL > $SUBTITLE_FILE
      1
      00:00:00,000 --> 00:00:02,000
      This is the red segment.

      2
      00:00:02,000 --> 00:00:04,000
      This is the green segment.

      3
      00:00:04,000 --> 00:00:06,000
      This is the blue segment.

      4
      00:00:06,000 --> 00:00:08,000
      This is the yellow segment.

      5
      00:00:08,000 --> 00:00:10,000
      End of the video.
      EOL
      ffmpeg -i $TMPFPATH -i $SUBTITLE_FILE -c:v copy -c:s mov_text -metadata:s:s:0 language=eng $FPATH
      rm $TMPFPATH
      rm $SUBTITLE_FILE
    '';

  # preprocess the videos
  scripts.generate-test-preprocessed.exec = ''
    go run --tags fts5 ./apps/clyper preprocess run testing/samplevideo testing/samplevideo-output
  '';

  # start the backend api server
  scripts.backend.exec = ''
    go run --tags fts5 ./apps/clyper serve \
      --objstore ./testing/samplevideo-output
  '';

  # Download thumbnail for S01E01 at timestamp 10ms
  scripts.sample-thumb.exec = ''
    curl 'localhost:8991/thumb/1/1/10' > sample-thumb.jpg
  '';

  # list nearby thumbnails of S01E01 at timestamp 10ms
  scripts.list-thumbs.exec = ''
    curl 'localhost:8991/thumbs/1/1/10' > sample-thumb.jpg
  '';

  # search for subs
  scripts.search.exec = ''
    curl 'localhost:8991/search?q="yellow"'
    curl 'localhost:8991/search?q="red"'
    curl 'localhost:8991/search?q="blue"'
  '';

  # get a gif
  scripts.make-gif.exec = ''
    curl 'http://localhost:8991/gif/1/1/1500/3000.gif?text=hello' > sample.gif
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
