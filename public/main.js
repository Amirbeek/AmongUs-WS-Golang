// --- Real-time Front (Among-Us style) ---

// App State
const state = {
    username: null,
    room: { code: 'A1', name: 'Skeld' },
    role: 'crew',
    phase: 'waiting',
    selfId: null,
    players: [
        { id: 'p1', name: 'Nova',  alive: true },
        { id: 'p2', name: 'Bolt',  alive: true },
        { id: 'p3', name: 'Echo',  alive: true },
        { id: 'p4', name: 'Pixel', alive: true },
        { id: 'p5', name: 'Astra', alive: true },
        { id: 'p6', name: 'Vanta', alive: true },
    ],
    messages: [
        { from: 'System', text: 'Welcome aboard! Discuss and find the killer…', self: false },
    ],
};

let lastSentText = null;
let lastSentAt = 0;
let conn = null;

// --- DOM Refs ---
const landing = document.getElementById('landing');
const app = document.getElementById('app');
const nameInput = document.getElementById('nameInput');
const enterBtn = document.getElementById('enterBtn');
const roomSelect = document.getElementById('roomSelect');
const customRoomWrap = document.getElementById('customRoomWrap');
const customRoom = document.getElementById('customRoom');
const roomBadge = document.getElementById('roomBadge');

const roleBadge = document.getElementById('roleBadge');
const playerList = document.getElementById('playerList');
const targetSelect = document.getElementById('targetSelect');
const actionBtn = document.getElementById('actionBtn');
const chatArea = document.getElementById('chatArea');
const msgInput = document.getElementById('msgInput');
const sendBtn = document.getElementById('sendBtn');
// const devRole = document.getElementById('devRole');
const mainPanel = document.getElementById('mainPanel');
const sidebar = document.getElementById('sidebar');

// --- Helpers ---
function sanitize(str){ return String(str).replace(/[<>]/g, s => ({'<':'&lt;','>':'&gt;'}[s])); }
function isOpen(){ return conn && conn.readyState === WebSocket.OPEN; }
function addMsg(from, text, self=false){
    state.messages.push({ from, text, self });
    renderChat();
}
function sendEnvelope(type, data){
    if (!isOpen()) return false;
    try {
        conn.send(JSON.stringify({ type, data }));
        return true;
    } catch (e) {
        console.warn('send failed', e);
        return false;
    }
}

// --- Rendering ---
function renderRole(){
    roleBadge.textContent = state.role === 'killer' ? 'Killer' : 'Crewmate';
    roleBadge.className = 'badge role ' + (state.role === 'killer' ? 'killer' : 'crew');

    if(state.role === 'killer'){
        actionBtn.textContent = 'Kill';
        mainPanel.classList.add('killer-accent');
        sidebar.classList.add('killer-glow');
    } else {
        actionBtn.textContent = 'Suspect';
        mainPanel.classList.remove('killer-accent');
        sidebar.classList.remove('killer-glow');
    }
}

function renderRoom(){
    roomBadge.textContent = `Room #${state.room.code}`;
}

function renderPlayers(){
    playerList.innerHTML = '';
    targetSelect.innerHTML = '<option value="" selected>Select a player…</option>';

    // Serverdan kelgan players bo‘lsa shuni ishlatamiz; bo‘lmasa fake ro‘yxat qoladi
    const list = (state.players && state.players.length) ? state.players : [];

    // “self”ni aniqlash: selfId bo‘lsa o‘sha, bo‘lmasa isim orqali
    const isSelf = (p) => state.selfId ? (p.id === state.selfId) : (p.name === state.username);

    const all = list.length
        ? list
        : [{ id: 'self', name: state.username, alive: true, self: true }, ...state.players];

    all.forEach((p, idx)=>{
        const row = document.createElement('div');
        row.className = 'player';

        const ava = document.createElement('div');
        ava.className = 'avatar';
        ava.style.background = ['#9b5de5','#00bbf9','#f15bb5','#fee440','#00f5d4','#f72585','#4cc9f0'][idx % 7];

        const name = document.createElement('div');
        name.className = 'pname';
        const you = isSelf(p) ? ' (you)' : '';
        name.textContent = (p.name || 'Unknown') + you;

        const st = document.createElement('div');
        st.className = 'status';
        st.textContent = (p.alive === false) ? 'Died' : 'Alive';

        row.appendChild(ava); row.appendChild(name); row.appendChild(st);
        playerList.appendChild(row);

        // Target ro‘yxatiga faqat “self” bo‘lmagan va alive bo‘lganlarni qo‘shamiz
        if(!isSelf(p) && (p.alive !== false)){
            const opt = document.createElement('option');
            opt.value = p.id || p.name || `p-${idx}`;
            opt.textContent = p.name || `Player ${idx+1}`;
            targetSelect.appendChild(opt);
        }
    });
}

function renderChat(){
    chatArea.innerHTML = '';
    state.messages.forEach(m=>{
        const line = document.createElement('div');
        line.className = 'msg';
        const bubble = document.createElement('div');
        const isSelf = m.self === true || (m.from && m.from === state.username);
        bubble.className = 'bubble' + (isSelf ? ' self' : '');
        const meta = document.createElement('div');
        meta.className = 'meta';
        meta.textContent = m.from || 'Unknown';
        const txt = document.createElement('div');
        txt.innerHTML = sanitize(m.text);
        bubble.appendChild(meta); bubble.appendChild(txt);
        line.appendChild(bubble);
        chatArea.appendChild(line);
    });
    chatArea.scrollTop = chatArea.scrollHeight;
}

function mountApp(){
    renderRole();
    renderRoom();
    renderPlayers();
    renderChat();
}

// --- Validation for Enter button ---
function validateEnter(){
    const nameOk = nameInput.value.trim().length > 0;
    const roomVal = roomSelect.value;
    const roomOk = roomVal !== 'custom' || customRoom.value.trim().length > 0;
    enterBtn.disabled = !(nameOk && roomOk);
}

// --- WebSocket ---
function connectWS(){
    try {
        const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
        const url = `${scheme}://${location.host}/ws?room=${encodeURIComponent(state.room.code)}&name=${encodeURIComponent(state.username)}`;
        conn = new WebSocket(url);

        conn.onopen = () => {
            console.log('WS connected:', url);
        };

        conn.onmessage = (ev) => {
            let asText = String(ev.data || '');

            try {
                const env = JSON.parse(asText);
                if (env && typeof env === 'object') {
                    const t = env.type;
                    const d = env.data || {};

                    switch (t) {
                        case 'hello': {
                            // {room, name} keladi
                            if (d.room) state.room.code = d.room;
                            addMsg('System', `Connected to room #${d.room || state.room.code} as ${d.name || state.username}`);
                            break;
                        }
                        case 'state': {
                            // {room, phase, gameEndsInSec, players:[{id,name,alive,ready}], you:{id,role}}
                            if (d.room) state.room.code = d.room;
                            if (d.phase) state.phase = d.phase;
                            if (Array.isArray(d.players)) state.players = d.players;
                            if (d.you) {
                                if (d.you.id) state.selfId = d.you.id;
                                if (d.you.role) state.role = d.you.role;
                            }
                            renderRole();
                            renderRoom();
                            renderPlayers();
                            break;
                        }
                        case 'phase': {
                            if (d.status) state.phase = d.status;
                            addMsg('System', `Phase: ${state.phase}`);
                            break;
                        }
                        case 'chat': {
                            // {from, text}
                            const from = d.from || 'Unknown';
                            const text = d.text || '';
                            const self = (from === state.username);
                            addMsg(from, text, self);
                            break;
                        }
                        case 'vote_start': {
                            addMsg('System', `Voting started (${d.endsInSec || 10}s)…`);
                            break;
                        }
                        case 'vote_end': {
                            addMsg('System', d.note || 'Voting ended.');
                            break;
                        }
                        case 'end': {
                            // {result: "..."}
                            addMsg('System', `Game ended: ${d.result || ''}`);
                            break;
                        }
                        default: {
                            // Unknown type — ko‘rsatib qo‘yamiz
                            addMsg('System', `Unknown event: ${t}`);
                        }
                    }
                    return;
                }
            } catch (e) {
                // JSON emas — oddiy matn sifatida ishlaymiz
            }
            addMsg('System', asText);
        };

        conn.onclose = () => {
            addMsg('System', 'Disconnected from server.');
        };

        conn.onerror = (e) => {
            console.warn('WS error', e);
        };
    } catch (e) {
        console.warn('WS init failed', e);
    }
}

fetch('/rooms')
    .then(res => res.json())
    .then(rooms => {
        rooms.forEach(room => {
            const opt = document.createElement('option');
            opt.value = room.id;
            opt.textContent = room.name;
            roomSelect.appendChild(opt);
        });
    })
    .catch(err => console.error('Error fetching rooms:', err));

// --- Events (Landing) ---
roomSelect.addEventListener('change', ()=>{
    if(roomSelect.value === 'custom'){
        customRoomWrap.classList.remove('hidden');
    } else {
        customRoomWrap.classList.add('hidden');
    }
    validateEnter();
});
customRoom.addEventListener('input', validateEnter);
nameInput.addEventListener('input', validateEnter);
enterBtn.addEventListener('click', ()=>{
    const name = nameInput.value.trim();
    if(!name) { nameInput.focus(); return; }

    let code = 'A1';
    let roomName = 'Skeld';
    if(roomSelect.value === 'custom'){
        code = sanitize(customRoom.value.trim()).slice(0,16) || 'X0';
        roomName = 'Custom';
    } else {
        const [c, n] = roomSelect.value.split('|');
        code = c; roomName = n;
    }

    state.username = name;
    state.room = { code, name: roomName };

    landing.classList.add('hidden');
    app.classList.add('visible');
    addMsg('System', `${sanitize(name)} joined ${roomName} (#${code}).`);
    mountApp();
    msgInput.focus();
    connectWS();
});
nameInput.addEventListener('keydown', (e)=>{ if(e.key==='Enter' && !enterBtn.disabled) enterBtn.click(); });

// --- Dev role switch (faqat UI) ---
// devRole.addEventListener('change', ()=>{
//     state.role = devRole.value === 'killer' ? 'killer' : 'crew';
//     renderRole();
// });

// --- Chat actions (REAL DATA) ---
sendBtn.addEventListener('click', ()=>{
    const val = msgInput.value.trim();
    if(!val) return;
    const ok = sendEnvelope('chat', { text: val, from: state.username });
    lastSentText = val;
    lastSentAt = Date.now();
    if (!ok) {
        addMsg(state.username || 'You', val, true);
    }

    msgInput.value='';
});

msgInput.addEventListener('keydown', (e)=>{ if(e.key==='Enter') sendBtn.click(); });

actionBtn.addEventListener('click', ()=>{
    const targetId = targetSelect.value;
    if(!targetId) return;

    if(state.role === 'killer'){
        // Kelajak: kill
        const ok = sendEnvelope('kill', { targetId, from: state.username });
        if (!ok) addMsg('System', `Attempted to kill ${targetId} (offline).`);
    } else {
        // Kelajak: vote/suspect
        const ok = sendEnvelope('vote', { targetId, from: state.username });
        if (!ok) addMsg('System', `Voted for ${targetId} (offline).`);
    }
});

validateEnter();
