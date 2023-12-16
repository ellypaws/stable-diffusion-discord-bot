# Stable Diffusion Discord Bot

<img src="https://go.dev/images/gophers/ladder.svg" width="48" alt="Go Gopher climbing a ladder." align="right">

An opinionated fork of https://github.com/pitapan5376/stable-diffusion-discord-bot

### Controlnet

Added an option to upload an attachment and have it process through controlnet.
Can also use img2img as the input.

添付ファイルをアップロードして、controlnetを介して処理するオプションを追加しました。
入力としてimg2imgも使用できます。

![controlnet](/document/020_controlnet.png?raw=true)

### Checkpoint

Added an option to select the checkpoint for each generation. This also has fuzzy search.
The checkpoint gets updated for upscaling and variation as well.

各世代のチェックポイントを選択するオプションを追加しました。これもファジー検索ができます。
チェックポイントは、アップスケーリングとバリエーションでも更新されます。

![checkpoint](/document/016_checkpoint.png?raw=true)

Also added it to the imagine settings command

imagine settingsコマンドにも追加しました

![imagine settings](/document/018_checkpoint_settings.png?raw=true)

### Img2Img

Added an option to upload an attachment and have it process through img2img.

添付ファイルをアップロードして、img2imgを介して処理するオプションを追加しました。

![img2img](/document/019_img2img.png?raw=true)

### Delete button for generation

Added a delete button to delete the message and the image. Only the user who generated the image can delete the message.

メッセージと画像を削除するための削除ボタンを追加。画像を生成したユーザーのみがメッセージを削除できます。

![delete button](/document/013_delete_button.png?raw=true)

### Extra image generation parameters

Added a lot of extra options for the imagine slash command for ease of use.

使いやすさのために imagine スラッシュコマンドのための多くの追加オプションを追加しました。

![extra parameters](/document/014_extra_options.png?raw=true)

### Lora fuzzy search

Added a fuzzy search for Lora models. You can now search for models by name, and the bot will return the closest match.
It also allows you to specify a weight by adding a colon after the name, followed by the weight. For example, `lora:0.5` will search for models with the name Lora and a weight of 0.5.
The last item will show you if the model can be found (you can click on it too)

Loraモデルのためのファジー検索を追加しました。名前でモデルを検索することができ、ボットは最も近い一致を返します。
名前の後にコロンを追加し、その後に重みを追加することで、重みを指定することもできます。例えば、「`lora:0.5`」は、名前がLoraで重みが0.5のモデルを検索します。
最後の項目は、モデルが見つかるかどうかを表示します（クリックすることもできます）

![lora fuzzy search](/document/015_lora.png?raw=true)

### ADetailer

Added an option to run the adetailer scripts on generation. We have a selection for Face, Body, or both.
This has been written as an interface to allow for more scripts to be added in the future.

生成時にadetailerスクリプトを実行するオプションを追加しました。Face、Body、または両方を選択できます。
これは、将来的により多くのスクリプトを追加できるようにするために、インターフェースとして書かれています。

![adetailer](/document/017_adetailer.png?raw=true)


### Updates

#### 1.3.7: [4e4d8f4](https://github.com/ellypaws/sd-discord-bot/commit/4e4d8f4d05caaf14511139069fd99f7920496cb1)

* Added raw json input to send to the API
* APIに送信するための生のJSON入力を追加しました
* Automatically revert model and lora after generation
* 生成後にモデルとロラを自動的に元に戻す

#### 1.3.6: [8abd059](https://github.com/ellypaws/sd-discord-bot/commit/8abd05988943940173230a007901055268cc9d0f)

* Interrupt button to cancel the current generation
* 現在の生成をキャンセルするための中断ボタン
* Cancel generation while in queue
* キュー中に生成をキャンセルする

#### 1.3.4: [3e5719e](https://github.com/ellypaws/sd-discord-bot/commit/3e5719eec95df990a2be22264a3613be9f3c5f16)

* Rewrite the output to display a Discord embed
* 出力を書き直してDiscord埋め込みを表示する

#### 1.2.3: [6fb9992](https://github.com/ellypaws/sd-discord-bot/commit/6fb9992eeaf93792833d95a69892e7d2e3042773)

* Added generic --parameter for key value pairs in prompt
* プロンプトのキーと値のペアのための一般的な--パラメータを追加しました

#### Under the hood

1. [x] Img2Img
2. [x] Controlnet
3. [x] Interrupt generation
4. [x] Moved handlers to a map to easily add and find them with constants
5. [x] Also did this for the components
6. [x] Added a progress bar while generating
7. [x] Methods for getting the current config and checkpoint
8. [x] Methods to get the available checkpoints and loras and store them in a cache slice
9. [x] Automatically use the face ADetailer even if the user doesn't specify it
10. [x] Allow changing vae and searching for hypernetwork models
11. [x] Command to reload lora, checkpoints, and vae
12. [x] Interrupt button and cancel from queue
13. [ ] Implement bubbletea TUI to include logging, progress bar, and api heartbeat
14. [ ] With bubbletea, add options to restart API, interrupt generations
15. [ ] Allow single image generations
16. [ ] Embed png info into the image

---

1. [x] Img2Img
2. [x] Controlnet
3. [x] 生成を中断する
4. [x] ハンドラーをマップに移動して、定数で簡単に追加したり見つけたりできるようにしました。
5. [x] コンポーネントにも同じことをしました。
6. [x] 生成中に進捗バーを追加しました。
7. [x] 現在の設定とチェックポイントを取得するためのメソッド
8. [x] 利用可能なチェックポイントとロラを取得し、キャッシュスライスに格納するためのメソッド
9. [x] ユーザーが指定しなくても、顔のADetailerを自動的に使用するようにしました。
10. [x] VAEを変更したり、ハイパーネットワークモデルを検索したりすることを許可する
11. [x] lora、チェックポイント、vaeを再読み込みするコマンド
12. [x] 中断ボタンとキューからのキャンセル
13. [ ] ロギング、進捗バー、およびAPIハートビートを含むbubbletea TUIを実装する
14. [ ] bubbleteaを使用して、APIの再起動、生成の中断などのオプションを追加します。
15. [ ] 単一の画像生成を許可する
16. [ ] png情報を画像に埋め込む

---

forked from https://github.com/AndBobsYourUncle/stable-diffusion-discord-bot
at 2023-04-08T07:10:00 (JST)

この時点のやつをフォークした。Go言語もDiscordクライアントも初なので分かるところだけ自分用に進める

I'm new on Go programming language and Discord client development.

Update: my version is not compatible with the upstream version, especially database columns.
        (if you have switched from his version, put my version to another folder.) 

## 001. changed button order and icon captions
at iPhone Discord client, result buttons was not lined up

iPhoneの Dicordクライアントで見づらかったのでボタン配置とキャプションを変更した

![button orders before](/document/001_change_button1.jpg?raw=true)

shorten caption and swap button orders like midjourney/nijijourney

![rearranged button order](/document/001_change_button2.jpg?raw=true)

## 002. changed fonts
prompt showing in monospace in Discord client

打ち間違いとかするのでプロンプトはMonospaceで表示するようにした

![change prompt font to monospace](/document/002_change_prompt_font.png?raw=true)


## 003. enable aspect ratio (without upscaler)
--ar param is parsed and computed new values, but actual result was still default 512x512. 
new width, height is fed to Image Generation param.
this is not using upscalers nor hires.fix, --ar 1:2 gives you 512x1024 so requires enough GPU memory.

ソース内にアスペクト比の機能が書いてあったけど512x512のまま動作しなかったのでなんか計算されてた値を渡せるようにした。
素の値なのでGPU必要（hires.fixとかupscaleはされてないっぽい）

### 1girl --ar 4:3
![sample for ar 4:3](/document/003_aspect_ratio_4_3.png?raw=true)

### 1girl --ar 1:2
![sample for ar 1:2](/document/003_aspect_ratio_1_2.png?raw=true)


## 004. add sampling steps
It seems difficult to add parameter like 'prompt:' on Discord ways, so I just copy & modified along --ar parameter.
with --step X gives you result in X steps processed. 
default value is 20, if not specified --step parameter. (sampling method is default Euler_a yet)

Discord風に prompt: とかでパラメータ渡すやり方が難しそうだったのでアスペクト比を取っているやつをコピーしてステップ数を渡せるようにした

### --step 7(512x512)
low steps at 7, seems noisy but It works.

![sample for step 7](/document/004_steps_param_7.png?raw=true)

### --step 50 --ar 2:1(1024x768)
combo with aspect ratio, I could make AI output more detailed result. 

アスペクト比を横長とかにしたとき、ステップ数を増やすことができるようになったので、画像が荒くなくなった。

![sample for ar 1:2](/document/004_steps_param_50.png?raw=true)

## 005. add CFG scale parameter
passing CFG scale value. 1.0(min.) to 30.0(max.); the limit is along with AUTO1111 WebUI.

--cfgscale で CFGスケール値を渡せるようにした。数値の許容範囲はAUTO1111のWebUIにあわせた。

image is generated by random seed at current, it's little bit hard to check effectiveness of CFG values.

seedの指定がまだできないので簡単には比較できないが、極端な値にしたら絵柄変わったので多分動いている。

### --cfgscale 1.2
![sample for CFG scale low](/document/005_cfg_scale_1.png?raw=true)

### --cfgscale 15.3
![sample for CFG scale high](/document/005_cfg_scale_15.png?raw=true)

## 006. seed parameter
passing seed parameter. default was -1(random); max value 12345678901234567890 is accepted by WebUI, but datatype of the code is int.
valid range of this program should be 0 - 2147483647.(Golang int)

seed値を渡せるようにした。StableDiffusionでの最大値を調べたが、はっきりしたことは不明だった。
実際にWebUIで適当に長い数字を入れると 12345678901234567890 までは受け付けて、桁を増やすとPythonのlong型でオーバーフローしているようだった。
Go言語の実装でint型になっているようなのでその値を最大とした。
実用上困らないと考えてがんばらないことにした。

### --seed 111
同じシード値で同じ結果が出ることが確認できた。

![sample for seed](/document/006_seed.png?raw=true)

### display seed value when upscaled
when no seed specified (at random), indicates clearly saying random

シード値未指定のときに -1 の表示になっていたので作成後メッセージで random(-1)表記にした

![indicate as random seed value](/document/006_seed2.png?raw=true)

when upscaling, display seed value at post-generate message

アップスケールのときに作成後メッセージに seed を表示するようにした

![show seed value at upscale](/document/006_seed3.png?raw=true)

## 007. negative_prompt: param

passing negative_prompt as optional parameter.
default value is hard-coded one: 
  "ugly, tiling, poorly drawn hands, poorly drawn feet, poorly drawn face, out of frame,
   mutation, mutated, extra limbs, extra legs, extra arms, disfigured, deformed, cross-eye,
   body out of frame, blurry, bad art, bad anatomy, blurred, text, watermark, grainy"

ネガティブプロンプトを与えられるようにした。

初期値（未指定）はハードコーディングされている上記の値とみなす。何か値を与えるとそれのみをネガティブプロンプトとする

![negative prompt param](/document/007_negative_prompt.png?raw=true)

最初は下のURLを参考に全部入れようとしたがうまくいかなかったので、まずネガティブプロンプトだけにした。

https://github.com/AndBobsYourUncle/stable-diffusion-discord-bot/pull/18/commits/cf6ec0d52461d0d2eaac2b5fd98316f88c14b43b


## 008. BUGFIX: seed value for big int

very few times SD api returns seed value as large int value which is over int32.

seed未指定で生成していると intの最大値を超えるものが来たときbotが落ちるのを確認した。
実害があったので直すことにした。

Go言語における最大値　符号付きint64　9223372036854775807 (2 ^ 63 - 1)

DBのSQLite3のint　　　INTEGERは、値に応じて 0～8byteの符号付き整数

ということで、データベースのアップグレード処理は書かなくても大丈夫だった。

![seed on bigint](/document/008_seed_bigint.png?raw=true)


## 009. selection pop-up for sampler
you can select a sampler for image generation (optional, default is Euler_a)

sampler指定ができるようにした。未指定のときはEuler_aが指定されたものとみなす。

サンプラーの順番は自分で使いやすいように並べた。

![sampler_choice](/document/009_sampler_selection.png?raw=true)

### Sampler: DPM++ S2 a Karras

` prompt: 1girl --seed 1 negative_prompt:EasyNegative sampler_name: DPM++ S2 a Karras `

![sampler_choice](/document/009_sampler_DPMppS2aKarras.png?raw=true)

### Sampler: DPM Adaptive

` prompt: 1girl --seed 1 negative_prompt:EasyNegative sampler_name: DPM Adaptive `

![sampler_choice](/document/009_sampler_DPMAdaptive.png?raw=true)

### Sampler: UniPC

` prompt: 1girl --seed 1 negative_prompt:EasyNegative sampler_name: UniPC `

![sampler_choice](/document/009_sampler_UniPC.png?raw=true)

## 010. Hires.fix
hires.fix に部分的に対応した。
hr_scale（拡大率） とhr_upscaler（アップスケーラー名）をテーブルに追加した。
縦長などの時の破綻がなくなった。

実際の拡大率指定は未実装。

EXEを再実行するとき、SQLiteのマイグレーションで項目が追加される。
なので、それ以前の結果を選択して再生成・アップスケールを行うと、該当項目がないのでエラーになる。
imagineコマンドのリンクをクリックしてDBから再度プロンプトなどの内容を取得して、作り直せばよい。

強制的にHiresFixはONにしてある。
拡大率は2にしてあるが、--ar の指定で自動計算された縦横サイズのまま出力される。（そこから拡大はされない）
WebUIの動作を見ると拡大率を指定するか、または完成サイズを指定するようになっている。
元ソースの作りのまま、--ar 指定に応じて完成サイズを固定している。

## 011. BUGFIX: NegativePrompt
NegativePrompt didn't applied, have fixed it.
ネガティブプロンプトが反映されていなかったのに気付いたので確認して直した。


## 012. Hires.fix with Zoom Rate (param: --zoom)
to switch hires.fix ON/OFF with Discord choice way. default: OFF for better generation performance.
you can specify the ratio with '--zoom x.x' (default: 2.0) param only if hires_fix=YES

APIに渡すパラメータを修正して、hires.fix をオプションでON-OFF できるようにした。
![hiresfix1](/document/012_hiresfix1.png?raw=true)

hires.fix のオプションをYESにしているとき、--zoom 1.2 のように指定すると元サイズから拡大される。

初期値は AUTO1111 WebUIにあわせて 2倍にした。

![hiresfix2](/document/012_hiresfix2.png?raw=true)
![hiresfix4](/document/imagine_20230624195929.jpg)

### hires.fix あり

![hiresfix3](/document/012_hiresfix3.png?raw=true)

### hires.fix なし
hires.fix のオプションが未指定、またはNOのときは、zoom指定をしても無視される。

![hiresfix4](/document/012_hiresfix4.png?raw=true)


## 013. apply upstream update
フォーク元が更新されていたので取り込んだ。バッチサイズが設定できるらしい。他に変更は特になし。（興味もない）

included upstream(AndBobsYourUncle's) update. It seems to change batch_size and program code improvements.
but I don't care about that.

