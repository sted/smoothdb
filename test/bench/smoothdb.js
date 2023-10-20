import http from "k6/http";

export default function() {
    
    const url = "http://localhost:8081/api/pgrest/projects?select=id,name,clients(*)";
    //const url = "http://localhost:8081/api/test_anon/t1";

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Accept-Profile': 'test',
            'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYW5vbnltb3VzIiwiaWQiOiIifQ.jppQJJyMyXuKOn8wlajnnAxZigC1CDGg8_54_otr7Gw'
        },
     };
    
    const res = http.get(url, params);
};

