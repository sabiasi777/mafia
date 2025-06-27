window.addEventListener("DOMContentLoaded", () => {

    const startButton = document.getElementById("startButton")
    const playerList = document.getElementById("playerList")
    const roleContainer = document.getElementById('roleContainer');
    const roleText = document.getElementById('roleText');
    const gameArea = document.getElementById('gameArea');
    const gameScreen = document.getElementById('gameScreen');
    const sendButton = document.getElementById("sendButton")
    const microphoneButton = document.getElementById("micButton")
    
    var roomCode
    var ws
    var stream
    var mediaRecorder
    var mediaSource
    var sourceBuffer
    var mediaElement
    
    var incomingAudioQueue = []
    var micMode = false
    var isMediaSourceOpen = false
    var isAppending = false
    var currentUserName = getCurrentUserNameFromURL()
    
    startButton.addEventListener('click', () => {
        roomCode = window.location.pathname.split("/").pop()
        fetch("http://localhost:8080/start", {
            method: "POST",
            headers: {
                'Content-Type': "application/json"
            },
            body: JSON.stringify({ roomCode, currentUserName })
        })
        .then(res => {
            if (!res.ok) throw new Error("Failed to start game")
            return res.json()
        })
        .then(data => {
            connect()
            const me = data.find(p => p.name === currentUserName)
            if (me) {
                startGameUI(me)
            }
        })
        .catch(err => console.error(err))
    })
    
    sendButton.addEventListener("click", () => {
        const input = document.getElementById("messageInput")
        if (input.value.trim()) {
            const messageData = {
                sender: currentUserName,
                content: input.value.trim(),
                timestamp: new Date().toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}),
                type: "text"
            }
            ws.send(JSON.stringify(messageData))
            input.value = ""
        }
    })
    
    document.getElementById("messageInput").addEventListener("keypress", (e) => {
        if (e.key === "Enter") {
            sendButton.click()
        }
    })
    
    microphoneButton.addEventListener("click", toggleMicrophone)
    function checkPlayerCount() {
        const playerCount = playerList.querySelectorAll("li").length
        const countElement = document.querySelector('.player-count');
        if (countElement) {
            countElement.textContent = playerCount;
        }
        startButton.disabled = playerCount < 4
        
        if (playerCount >= 4) {
            startButton.innerHTML = '🎮 Start Game';
        } else {
            startButton.innerHTML = `🎮 Need ${4 - playerCount} more players`;
        }
    }
    
    checkPlayerCount()
    const observer = new MutationObserver(checkPlayerCount)
    observer.observe(playerList, { childList: true })

    function getCurrentUserNameFromURL() {
        const params = new URLSearchParams(window.location.search)
        return params.get("user")
    }
    
    function startGameUI(me) {
        document.getElementById("gameInfo").style.display = "none"
        document.getElementById("statusBar").style.display = "none"
        document.getElementsByTagName("header")[0].style.display = "none"
        startButton.style.display = "none"
        roleText.textContent = `Your role: ${me.role}`
        roleContainer.style.display = "flex"
        setTimeout(() => {
            roleContainer.style.display = "none"
            gameArea.style.display = "grid"
        }, 3000)
    }
    
    async function toggleMicrophone() {
        if (!stream || !mediaRecorder) {
            await setupAudio()
        }
        micMode = !micMode
        if (micMode) {
            mediaRecorder.start(10)
            console.log("Live streaming started")
            microphoneButton.className = "control-button mic-button unmuted"
            const playerContainer = document.querySelector(`.video-container[data-player-name="${currentUserName}"]`);
            if (playerContainer) {
                playerContainer.classList.add("speaking");
            }
            microphoneButton.textContent = "🎙️"
            startMediaSourcePlayback();
        } else {
            mediaRecorder.stop()
            console.log("Live streaming stopped")
            microphoneButton.className = "control-button mic-button muted"
            const playerContainer = document.querySelector(`.video-container[data-player-name="${currentUserName}"]`);
            if (playerContainer) {
                playerContainer.classList.remove("speaking");
            }
            microphoneButton.textContent = "🔇"
            stopMediaSourcePlayback();
        }
    }
    
    function connect() {
        ws = new WebSocket(`ws://localhost:8080/ws/chat?room=${roomCode}`)
        ws.binaryType = "arraybuffer"
    
        ws.onopen = function() {
            console.log("Connected to WebSocket server")
        }
    
        ws.onmessage = async function(event) {
            if (typeof event.data === 'string') {
                const message = JSON.parse(event.data)
                console.log("Text message received (likely chat):", message)
                const messageDisplay = document.getElementById("messages")
                const messageElement = document.createElement("div")
                messageElement.className = "message"
                messageElement.innerHTML = `
                    <div class="message-sender">${message.sender}</div>
                    <div class="message-content">${message.content}</div>
                    <div class="message-time">${message.timestamp}</div>
                `
                messageDisplay.appendChild(messageElement)
                messageDisplay.scrollTop = messageDisplay.scrollHeight
            } else if (event.data instanceof ArrayBuffer) {
                console.log("Audio message received!!")
                const audioBlob = new Blob([event.data], { type: 'audio/webm;codecs=opus' })
                incomingAudioQueue.push(audioBlob)
                processIncomingAudioQueue()
            } else {
                console.warn("Received unknown message type from WebSocket:", typeof event.data, event.data)
            }
        }
    
        ws.onclose = function() {
            console.log("WebSocket connection closed, retrying...")
            setTimeout(connect, 1000)
        }
    
        ws.onerror = function(error) {
            console.error("WebSocket error:", error)
        }
    }
    
    async function setupAudio() {
        if (!stream) {
            stream = await navigator.mediaDevices.getUserMedia({ audio: true })
            }
        if (!mediaRecorder) {
            mediaRecorder = new MediaRecorder(stream, { mimeType: "audio/webm;codecs=opus" })
            mediaRecorder.ondataavailable = async (e) => {
                if (e.data.size > 0) {
                    console.log("Sending audio chunk with type:", e.data.type, "and size:", e.data.size)
                    fetch("http://localhost:8080/audio", {
                        method: "POST",
                        headers: {
                            "Room-Code": roomCode,
                            "Content-Type": e.data.type,
                            'X-Mime-Type': e.data.type
                        },
                        body: e.data
                    })
                    .catch(err => console.error("Error sending audio chunk:", err))
                }
            }
        }
    }
    
    function startMediaSourcePlayback() {
        if (mediaElement) {
            stopMediaSourcePlayback()
        }
        mediaElement = document.getElementById("liveAudioPlayer");
        if (!mediaElement) {
            mediaElement = document.createElement("audio");
            mediaElement.id = "liveAudioPlayer";
            mediaElement.autoplay = true
            document.body.appendChild(mediaElement)
        } else {
            mediaElement.src = "";
            mediaElement.autoplay = true;
        }
        mediaSource = new MediaSource();
        mediaElement.src = URL.createObjectURL(mediaSource);
        mediaSource.addEventListener('sourceopen', () => {
            console.log("MediaSource opened.");
            isMediaSourceOpen = true;
            try {
                const mimeCodec = 'audio/webm; codecs="opus"';
                if (!MediaSource.isTypeSupported(mimeCodec)) {
                    console.error('MIME type not supported:', mimeCodec);
                    alert('Your browser does not support WebM Opus audio playback for MediaSource. Live audio will not work.');
                    return;
                }
                sourceBuffer = mediaSource.addSourceBuffer(mimeCodec);
                console.log("SourceBuffer added.");
                sourceBuffer.addEventListener('updateend', () => {
                    isAppending = false;
                    processIncomingAudioQueue()
                });
                sourceBuffer.addEventListener('error', (e) => {
                    console.error('SourceBuffer error:', e);
                });
            } catch (e) {
                console.error("Error adding SourceBuffer:", e);
                isMediaSourceOpen = false;
            }
            processIncomingAudioQueue()
        });
        mediaSource.addEventListener('sourceended', () => {
            console.log("MediaSource ended.");
            isMediaSourceOpen = false;
        });
        mediaSource.addEventListener('sourceclose', () => {
            console.log("MediaSource closed.");
            isMediaSourceOpen = false;
        });
    }
    
    function stopMediaSourcePlayback() {
        if (mediaElement) {
            mediaElement.pause();
            mediaElement.src = "";
        }
        if (mediaSource && mediaSource.readyState === 'open') {
            try {
                mediaSource.endOfStream();
            } catch (e) {
                console.warn("Error ending MediaSource stream:", e);
            }
        }
        mediaSource = null;
        sourceBuffer = null;
        incomingAudioQueue = [];
        isMediaSourceOpen = false;
        isAppending = false;
        console.log("MediaSource playback stopped and cleared.");
    }
    
    function processIncomingAudioQueue() {
        if (!isMediaSourceOpen || !sourceBuffer || isAppending || incomingAudioQueue.length === 0) {
            return
        }
        const audioBlob = incomingAudioQueue.shift();
        if (audioBlob) {
            isAppending = true;
            console.log("Appending audio data to SourceBuffer. Size:", audioBlob.size);
            audioBlob.arrayBuffer().then(buffer => {
                try {
                    sourceBuffer.appendBuffer(buffer);
                } catch (e) {
                    console.error("Error appending buffer:", e);
                    isAppending = false
                }
            }).catch(err => {
                console.error("Error converting blob to arrayBuffer for append:", err);
                isAppending = false;
            });
        }
    }
})